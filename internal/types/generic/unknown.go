package generic

import "reflect"

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
