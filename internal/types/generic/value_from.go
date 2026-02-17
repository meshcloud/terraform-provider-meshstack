package generic

import (
	"fmt"
	"reflect"
	"slices"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-go/tftypes"

	reflectwalk "github.com/meshcloud/terraform-provider-meshstack/internal/util/reflect"
)

// ValueFrom builds a Terraform value from the given generic input using reflection.
// It traverses nil values to construct the full Terraform type for the value.
// The value itself can be null if the input is nil.
// Adjustments are made with WithValueFromConverter option.
func ValueFrom[T any](in T, opts ...ConverterOption) (tftypes.Value, error) {
	return ConverterOptions(opts).newConverter().
		valueFrom(reflect.ValueOf(in), nil, false, false)
}

// WithValueFromEmptyContainer defines a handler which is called when the value encountered is an empty slice or map.
// By default, empty containers are converted to empty Terraform values (not null), and the handler can change this to a null value if desired.
func WithValueFromEmptyContainer(handler ValueFromEmptyContainerHandler) ConverterOption {
	return func(c *converter) {
		c.ValueFromEmptyContainer = handler
	}
}

// WithUseSetForElementsOf detects slices with given T as sets and uses the corresponding [tftypes.Set] for building the value.
func WithUseSetForElementsOf[T any]() ConverterOption {
	return func(c *converter) {
		c.SetElemTypes = append(c.SetElemTypes, reflect.TypeFor[T]())
	}
}

// WithValueFromConverter set additional converters for ValueFrom. See also WithValueFromConverterFor for more convenience.
func WithValueFromConverter(converters ...ValueFromConverter) ConverterOption {
	return func(c *converter) {
		c.ValueFromConverters = append(c.ValueFromConverters, converters...)
	}
}

// WithValueFromConverterFor converts a target type T with the given ValueFromFunc mapper.
// Supports dedicated nilness handling with ValueFromConverterForTypedNilHandler.
func WithValueFromConverterFor[T any](nilHandler func() (tftypes.Value, error), f ValueFromFunc[T]) ConverterOption {
	targetType := reflect.TypeFor[T]()
	return WithValueFromConverter(func(attributePath path.Path, in reflect.Value, haveNil, haveUnknown bool) (out tftypes.Value, matched bool, err error) {
		if !in.Type().AssignableTo(targetType) {
			return // matched=false
		}
		matched = true
		// Handle nullability properly here,
		// now we know the targetType thanks to generics
		if haveNil || haveUnknown {
			if nilHandler == nil {
				// Use empty representation of T for nil values
				nilHandler = ValueFromConverterForTypedNilHandler[T]()
			}
			out, err = nilHandler()
			if err == nil && haveUnknown {
				out = tftypes.NewValue(out.Type(), tftypes.UnknownValue)
			}
			return
		}
		if targetValue, ok := in.Interface().(T); ok {
			out, err = f(attributePath, targetValue)
		} else {
			// this should never happen, as we've checked conversion above!
			panic(fmt.Sprintf("attribute path %s: cannot convert %s, value %#v, to generic %T", attributePath, in.Type(), in.Interface(), targetValue))
		}
		return
	})
}

// ValueFromConverterForTypedNilHandler constructs are nil value factory for given type T.
// Used when converter in WithValueFromConverterFor would otherwise get a nil/zero value passed down,
// which saves some repetitive work in the ValueFromFunc.
func ValueFromConverterForTypedNilHandler[T any]() func() (tftypes.Value, error) {
	return func() (tftypes.Value, error) {
		return ValueFrom[*T](nil)
	}
}

// See NullIsUnknown wrapper.
type unknowable interface {
	IsUnknown() bool
	Unwrap() reflect.Value
}

func (conv converter) valueFrom(in reflect.Value, path reflectwalk.WalkPath, haveNil, haveUnknown bool) (out tftypes.Value, err error) {
	defer func() {
		err = path.WrapError(err)
	}()

dereference:
	for {
		switch kind := in.Kind(); kind {
		case reflect.Ptr:
			if in.IsNil() {
				haveNil = in.IsNil()
				in = reflect.Zero(in.Type().Elem())
			} else {
				in = in.Elem()
			}
		default:
			break dereference
		}
	}

	if in.IsValid() {
		if u, ok := in.Interface().(unknowable); ok {
			return conv.valueFrom(u.Unwrap(), path, haveNil, u.IsUnknown())
		}
	}

	if len(conv.ValueFromConverters) > 0 {
		attributePath := conv.walkPathToAttributePath(path)
		for _, fromConverter := range conv.ValueFromConverters {
			if outValue, matched, err := fromConverter(attributePath, in, haveNil, haveUnknown); matched {
				return outValue, err
			}
		}
	}

	kind := in.Kind()
	newValueFrom := func(valueType tftypes.Type, values any) tftypes.Value {
		switch {
		case haveUnknown:
			return tftypes.NewValue(valueType, tftypes.UnknownValue)
		case haveNil:
			// This handles the famous Go any(nil) != other(nil) behavior for all values container types below.
			// [tftypes.NewValue] can now correctly check for nilness of the given 'any' value input.
			return tftypes.NewValue(valueType, nil)
		default:
			return tftypes.NewValue(valueType, values)
		}
	}

	switch kind {
	case reflect.Bool:
		return newValueFrom(tftypes.Bool, in.Bool()), nil
	case reflect.String:
		return newValueFrom(tftypes.String, in.String()), nil
	case reflect.Int64:
		return newValueFrom(tftypes.Number, in.Int()), nil
	case reflect.Slice:
		haveNil = haveNil || in.IsNil()
		if in.Len() == 0 && conv.ValueFromEmptyContainer != nil {
			haveNil, err = conv.ValueFromEmptyContainer(conv.walkPathToAttributePath(path))
			if err != nil {
				return
			}
		}
		values := make([]tftypes.Value, 0)
		var elemType tftypes.Type
		if err := path.WalkSlice(in, func(path reflectwalk.WalkPath, idx *reflectwalk.SliceIndex, in reflect.Value) error {
			sliceValue, err := conv.valueFrom(in, path, haveNil, haveUnknown)
			if err != nil {
				return err
			}
			if elemType == nil {
				elemType = sliceValue.Type()
			} else if !elemType.Equal(sliceValue.Type()) {
				return fmt.Errorf("non-unique element types encountered in slice: %s vs. %s", elemType, sliceValue.Type())
			}
			if !idx.EmptyContainer() {
				values = slices.Insert(values, int(*idx), sliceValue)
			}
			return path.Stop()
		}, reflectwalk.VisitEmptyContainers()); err != nil {
			return out, err
		}
		for _, setElemType := range conv.SetElemTypes {
			if setElemType.AssignableTo(in.Type().Elem()) {
				return newValueFrom(tftypes.Set{ElementType: elemType}, values), nil
			}
		}
		return newValueFrom(tftypes.List{ElementType: elemType}, values), nil
	case reflect.Map:
		haveNil = haveNil || in.IsNil()
		if in.Len() == 0 && conv.ValueFromEmptyContainer != nil {
			haveNil, err = conv.ValueFromEmptyContainer(conv.walkPathToAttributePath(path))
			if err != nil {
				return
			}
		}
		values := map[string]tftypes.Value{}
		var elemType tftypes.Type
		if err := path.WalkMap(in, func(path reflectwalk.WalkPath, mapKey *reflectwalk.MapKey, in reflect.Value) error {
			mapValue, err := conv.valueFrom(in, path, haveNil, haveUnknown)
			if err != nil {
				return err
			}
			if elemType == nil {
				elemType = mapValue.Type()
			} else if !elemType.Equal(mapValue.Type()) {
				return fmt.Errorf("non-unique element types encountered in map: %s vs. %s", elemType, mapValue.Type())
			}
			if !mapKey.EmptyContainer() {
				values[mapKey.Name()] = mapValue
			}
			return path.Stop()
		}, reflectwalk.VisitEmptyContainers()); err != nil {
			return out, err
		}
		return newValueFrom(tftypes.Map{ElementType: elemType}, values), nil
	case reflect.Struct:
		values := map[string]tftypes.Value{}
		types := map[string]tftypes.Type{}
		if err := path.WalkStruct(in, func(path reflectwalk.WalkPath, field *reflectwalk.StructField, _ reflect.Value) error {
			tfsdkTag := field.Tag.Get("tfsdk")
			if tfsdkTag == "-" {
				// explicitly ignored struct field
				return path.Stop()
			} else if strings.TrimSpace(tfsdkTag) == "" {
				return fmt.Errorf("tfsdk tag is required on struct %T, field %s", in.Interface(), field.Name)
			}
			vField, err := in.FieldByIndexErr(field.Index)
			if err != nil {
				return fmt.Errorf("field %s in struct %T is part of an embedded struct which is a nil pointer: %w", field.Name, in.Interface(), err)
			}
			fieldValue, err := conv.valueFrom(vField, path, haveNil, haveUnknown)
			if err != nil {
				return err
			}
			values[tfsdkTag] = fieldValue
			types[tfsdkTag] = fieldValue.Type()
			return path.Stop()
		}, reflectwalk.VisitEmbeddedNilStructs()); err != nil {
			return out, err
		}
		return newValueFrom(tftypes.Object{AttributeTypes: types}, values), nil
	default:
		panic(fmt.Sprintf("kind %s not supported", kind))
	}
}

type ValueFromFunc[T any] func(attributePath path.Path, in T) (tftypes.Value, error)

// A ValueFromConverter gets haveNil=true if conversion happens below a nil container such as slice, map or pointer-to-struct.
// An implementation is expected to return a null tftypes.Value with proper type information.
type ValueFromConverter func(attributePath path.Path, in reflect.Value, haveNil, haveUnknown bool) (out tftypes.Value, matched bool, err error)
