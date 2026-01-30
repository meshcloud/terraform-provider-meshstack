package variant

import (
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
)

// A Variant represents a single JSON map entry having two different Go type representations X and Y.
// After JSON unmarshalling you can check with HasX, HasY which field has been detected, while X is preferred.
// An example usage is a Client DTO response which can either be struct representing a secret hash,
// or a simple string response if that's a non-sensitive value.
type Variant[X, Y any] struct {
	X X
	Y Y
}

var (
	_ json.Unmarshaler = (*Variant[int, string])(nil)
	_ json.Marshaler   = Variant[int, string]{}
)

func (v Variant[X, Y]) MarshalJSON() ([]byte, error) {
	if v.HasX() {
		return json.Marshal(v.X)
	} else if v.HasY() {
		return json.Marshal(v.Y)
	} else {
		return json.Marshal(nil)
	}
}

func (v Variant[X, Y]) HasX() bool {
	x := reflect.ValueOf(v.X)
	return x.IsValid() && !x.IsZero()
}

func (v Variant[X, Y]) HasY() bool {
	y := reflect.ValueOf(v.Y)
	return y.IsValid() && !y.IsZero()
}

func (v Variant[X, Y]) WithX(action func(x *X)) {
	if v.HasX() {
		action(&v.X)
	} else {
		action(nil)
	}
}

func (v Variant[X, Y]) WithY(action func(y *Y)) {
	if v.HasY() {
		action(&v.Y)
	} else {
		action(nil)
	}
}

func (v *Variant[X, Y]) UnmarshalJSON(bytes []byte) error {
	errX := json.Unmarshal(bytes, &v.X)
	errY := json.Unmarshal(bytes, &v.Y)
	switch {
	case v.HasX() && v.HasY():
		// Explicitly prefer X over Y and set Y to zero even if unmarshalling has also worked,
		// this supports having Y with catch-all type 'any'
		var zeroY Y
		v.Y = zeroY
		return errX
	case v.HasX():
		return errX
	case v.HasY():
		return errY
	default:
		var nothing any
		if err := json.Unmarshal(bytes, &nothing); err != nil {
			return fmt.Errorf("cannot unmarshal to any: %w", err)
		}
		if nothing == nil {
			// support optional unmarshalling aka neither X nor Y is set
			return nil
		}
		return errors.Join(fmt.Errorf("variant[%T, %T]: cannot unmarshal '%s' to any field", v.X, v.Y, string(bytes)), errX, errY)
	}
}
