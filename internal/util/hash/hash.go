package hash

import (
	"crypto/sha256"
	"crypto/subtle"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"hash"
	"reflect"

	reflectwalk "github.com/meshcloud/terraform-provider-meshstack/internal/util/reflect"
)

type Hasher struct {
	Underlying func() hash.Hash

	// Private fields are modified by Option passed to Hasher.Hash.
	disallowedMapKeys []reflect.Value
}

type Sum []byte

type Option func(*Hasher)

func (h Hasher) Hash(v any, opts ...Option) (Sum, error) {
	if h.Underlying == nil {
		h.Underlying = sha256.New
	}
	for _, opt := range opts {
		opt(&h)
	}
	sum, _, err := h.hash(nil, reflect.ValueOf(v))
	return sum, err
}

func (s Sum) Hex() string {
	return hex.EncodeToString(s)
}

func DisallowMapKeys[T comparable](keys ...T) Option {
	return func(h *Hasher) {
		for _, key := range keys {
			h.disallowedMapKeys = append(h.disallowedMapKeys, reflect.ValueOf(key))
		}
	}
}

func (h Hasher) hash(root reflectwalk.WalkPath, v reflect.Value) (Sum, bool, error) {
	w := writer{hasher: h.Underlying()}
	visitor := func(path reflectwalk.WalkPath, v reflect.Value) error {
		kind := v.Kind()

		if kind == reflect.Ptr || kind == reflect.Interface {
			if v.IsNil() {
				return path.Stop()
			} else {
				return nil
			}
		}

		switch kind {
		case reflect.Slice, reflect.Array:
			if v.Len() == 0 {
				return path.Stop()
			}
			if err := w.WriteBinary(uint64(kind)); err != nil {
				return err
			}
			var total Sum
			if err := path.WalkSlice(v, func(path reflectwalk.WalkPath, value reflect.Value) (err error) {
				err = path.Stop() // always stop walking further, as we hash value only (also recursively)
				if sum, written, err := h.hash(path, value); err != nil {
					return err
				} else if written {
					total.addIgnoringOrder(sum)
				}
				return
			}); err != nil {
				return err
			}
			if err := w.Write(total); err != nil {
				return err
			}
			return path.Stop()
		case reflect.Map:
			if v.Len() == 0 {
				return path.Stop()
			}
			if err := w.WriteBinary(uint64(kind)); err != nil {
				return err
			}
			var keyTotal, valueTotal Sum
			if err := path.WalkMap(v, func(path reflectwalk.WalkPath, key *reflectwalk.MapKey, value reflect.Value) (err error) {
				err = path.Stop() // always stop walking further, as we hash value only (also recursively)
				if sum, written, errHash := h.hash(path, value); errHash != nil {
					return errHash
				} else if written {
					valueTotal.addIgnoringOrder(sum)
				} else {
					// entirely ignore map entries which have "empty" value
					// do not hash the key, even don't check if disallowed
					return
				}
				// as we're stopping WalkMap immediately (as we've walked the map values via h.hash(value) above)
				// the provided key pointer should never be nil (otherwise reflectwalk has an implementation bug)
				if err := h.checkDisallowedMapKeys(path, key.Value); err != nil {
					return err
				}
				if keySum, _, err := h.hash(path, key.Value); err != nil {
					return err
				} else {
					keyTotal.addIgnoringOrder(keySum)
				}
				return
			}); err != nil {
				return err
			}
			if err := w.Write(keyTotal, valueTotal); err != nil {
				return err
			}
			return path.Stop()
		default:
			// continue with simpler kinds
		}

		if err := w.WriteBinary(uint64(kind)); err != nil {
			return err
		}
		switch {
		case v.CanFloat():
			return w.WriteBinary(v.Float())
		case kind == reflect.String:
			return w.Write([]byte(v.String()))
		case kind == reflect.Bool:
			return w.WriteBinary(v.Bool())
		case v.CanInt():
			return w.WriteBinary(v.Int())
		case v.CanUint():
			return w.WriteBinary(v.Uint())
		case v.CanComplex():
			return w.WriteBinary(v.Complex())
		}
		return fmt.Errorf("cannot handle kind %s", kind)
	}
	err := reflectwalk.Walk(v, visitor, reflectwalk.WithRoot(root))
	return w.hasher.Sum(nil), w.written, err
}

func (h Hasher) checkDisallowedMapKeys(path reflectwalk.WalkPath, key reflect.Value) error {
	for _, mapKey := range h.disallowedMapKeys {
		if key.Equal(mapKey) {
			return fmt.Errorf("key path %s matches one of disallowed keys %s", path, h.disallowedMapKeys)
		}
	}
	return nil
}

func (s *Sum) addIgnoringOrder(other Sum) {
	if len(*s) < len(other) {
		*s = append(*s, other[len(*s):]...)
		return
	} else {
		subtle.XORBytes(*s, *s, other)
	}
}

type writer struct {
	hasher  hash.Hash
	written bool
}

func (w *writer) Write(data ...[]byte) error {
	for _, p := range data {
		if n, err := w.hasher.Write(p); err != nil {
			return err
		} else if n > 0 {
			w.written = true
		}
	}
	return nil
}

func (w *writer) WriteBinary(v any) (err error) {
	if err := binary.Write(w.hasher, binary.LittleEndian, v); err != nil {
		return err
	} else {
		w.written = true
		return nil
	}
}
