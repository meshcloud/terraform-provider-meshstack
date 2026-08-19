package generic

import (
	"fmt"
	"reflect"

	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

// AttributeHasUnknown reports whether the named top-level attribute of raw holds an unknown value at
// any depth. Terraform marks an attribute unknown when it is wired to a resource the plan creates or
// replaces, and [ValueTo] cannot represent that, so a caller that would otherwise convert a planned
// value uses this to bail out first. It walks the raw value because an unknown can sit arbitrarily
// deep: an object is known while one of its nested attributes is not.
func AttributeHasUnknown(raw tftypes.Value, attributeName string) (bool, error) {
	found, _, err := tftypes.WalkAttributePath(raw, tftypes.NewAttributePath().WithAttributeName(attributeName))
	if err != nil {
		return false, fmt.Errorf("cannot resolve attribute %q: %w", attributeName, err)
	}

	value, ok := found.(tftypes.Value)
	if !ok {
		return false, fmt.Errorf("expected a value at attribute %q, got %T", attributeName, found)
	}

	unknown := false
	if err := tftypes.Walk(value, func(_ *tftypes.AttributePath, nested tftypes.Value) (bool, error) {
		if !nested.IsKnown() {
			unknown = true
		}
		return !unknown, nil
	}); err != nil {
		return false, fmt.Errorf("cannot walk attribute %q: %w", attributeName, err)
	}

	return unknown, nil
}

// NullIsUnknown wraps a non-nil known value or if nil,
// the value will become unknown during ValueFrom conversion.
// This can be more convenient compared to directly manipulating attributes.
type NullIsUnknown[T any] struct {
	Value *T
}

func (v *NullIsUnknown[T]) UnwrapAddr() reflect.Value {
	// Be careful that one is returning an addressable value even if the Value struct field is nil!
	return reflect.ValueOf(v).Elem().FieldByName("Value")
}

func (v NullIsUnknown[T]) IsUnknown() bool {
	return v.Value == nil
}

func (v NullIsUnknown[T]) Unwrap() reflect.Value {
	return reflect.ValueOf(v.Value)
}

func (v NullIsUnknown[T]) Get() T {
	return *v.Value
}

func KnownValue[T any](v T) NullIsUnknown[T] {
	return NullIsUnknown[T]{Value: &v}
}

var (
	// Make sure we can detect this NullIsUnknown wrapper in ValueTo, ValueFrom conversion.
	_ unknowable     = NullIsUnknown[any]{}
	_ unknowableAddr = &NullIsUnknown[any]{}
)
