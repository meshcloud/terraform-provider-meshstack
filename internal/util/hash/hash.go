package hash

import (
	"crypto/sha256"
	"crypto/subtle"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"hash"
	"io"
	"reflect"
	"slices"
	"strings"
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

func (h Hasher) hash(path traversalPath, v reflect.Value) (sum []byte, empty bool, err error) {
	hasher := h.Underlying()
	if empty, err = h.deepHash(path, binaryWriter{hasher}, v); err != nil {
		return nil, false, err
	} else {
		return hasher.Sum(nil), empty, nil
	}
}

func (h Hasher) deepHash(path traversalPath, w binaryWriter, v reflect.Value) (empty bool, err error) {
dereference:
	for {
		kind := v.Kind()
		switch kind {
		case reflect.Pointer, reflect.Interface:
			if v.IsNil() {
				return true, nil
			}
			path = path.AddPtr()
			v = v.Elem()
			continue
		default:
			break dereference
		}
	}
	if !v.IsValid() {
		return true, nil
	}
	// write the kind to detect changes in type if byte structure of value is the same
	// even for empty values this should matter
	kind := v.Kind()
	if err := w.WriteBinary(uint64(kind)); err != nil {
		return false, err
	}
	return false, h.deepHashNonEmpty(path, w, v)
}

func (h Hasher) deepHashNonEmpty(path traversalPath, w binaryWriter, v reflect.Value) error {
	kind := v.Kind()
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

	switch kind {
	case reflect.Slice, reflect.Array:
		var total Sum
		for i, elem := range v.Seq2() {
			if sum, isNil, err := h.hash(path.AddIndex(i), elem); err != nil {
				return err
			} else if !isNil {
				total.addIgnoringOrder(sum)
			}
		}
		return w.Write(total)
	case reflect.Map:
		var keyTotal, valueTotal Sum
		for key, value := range v.Seq2() {
			if valueSum, isNil, err := h.hash(path.AddKey(key), value); err != nil {
				return err
			} else if isNil {
				// ignore map entries which point to nil (even if disallowed)
				continue
			} else {
				valueTotal.addIgnoringOrder(valueSum)
			}
			if err := h.checkDisallowedMapKeys(path, key); err != nil {
				return err
			}

			if keySum, _, err := h.hash(path, key); err != nil {
				return err
			} else {
				keyTotal.addIgnoringOrder(keySum)
			}
		}
		return w.Write(keyTotal, valueTotal)
	default:
		return fmt.Errorf("cannot handle kind %s", kind)
	}
}

func (h Hasher) checkDisallowedMapKeys(path traversalPath, key reflect.Value) error {
	for _, mapKey := range h.disallowedMapKeys {
		if key.Equal(mapKey) {
			return fmt.Errorf("key '%s' must not be in disallowed keys %s", path.AddKey(key), h.disallowedMapKeys)
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

type binaryWriter struct {
	io.Writer
}

func (w binaryWriter) Write(data ...[]byte) error {
	for _, p := range data {
		if _, err := w.Writer.Write(p); err != nil {
			return err
		}
	}
	return nil
}

func (w binaryWriter) WriteBinary(v any) (err error) {
	return binary.Write(w.Writer, binary.LittleEndian, v)
}

type traversalPath []string

func (path traversalPath) String() string {
	return strings.Join(path, "")
}

func (path traversalPath) AddKey(key reflect.Value) traversalPath {
	return append(path, fmt.Sprintf(".%s", key))
}

func (path traversalPath) AddIndex(i reflect.Value) traversalPath {
	return append(path, fmt.Sprintf("[%d]", i.Int()))
}

func (path traversalPath) AddPtr() traversalPath {
	if l := len(path); l == 0 {
		return traversalPath{"*"}
	} else {
		clone := slices.Clone(path)
		if strings.HasPrefix(path[l-1], ".") {
			clone[l-1] = ".*" + path[l-1][1:]
		} else {
			clone[l-1] = "*" + path[l-1]
		}
		return clone
	}
}
