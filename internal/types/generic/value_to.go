package generic

import (
	"fmt"
	"maps"
	"math/big"
	"reflect"
	"slices"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-go/tftypes"

	reflectwalk "github.com/meshcloud/terraform-provider-meshstack/internal/util/reflect"
)

// ValueTo converts a Terraform value to the desired Go value of type T (which is typically a model struct with `tfsdk` tags).
// Adjustments are made with WithValueToConverter option.
// See also ValueFrom counterpart.
func ValueTo[T any](in tftypes.Value, opts ...ConverterOption) (out T, err error) {
	conv := ConverterOptions(opts).newConverter()
	// Have some default converters for primitive types (appended to make them the lowest precedence)
	conv.ValueToConverters = append(conv.ValueToConverters,
		valueToConverterMatchingKind[bool]{convertValueTo[bool], conv.SetUnknownValueToZero}.Adapt(),
		valueToConverterMatchingKind[string]{convertValueTo[string], conv.SetUnknownValueToZero}.Adapt(),
		valueToConverterMatchingKind[int64]{convertValueToInt64, conv.SetUnknownValueToZero}.Adapt(),
	)
	err = conv.valueTo(in, reflect.ValueOf(&out), nil)
	return
}

// WithSetUnknownValueToZero set the output to zero value (see reflect.Zero) if an unknown value is encountered in the input.
// Useful when converting input plans which have computed values set to unknown.
func WithSetUnknownValueToZero() ConverterOption {
	return func(c *converter) {
		c.SetUnknownValueToZero = true
	}
}

// WithValueToConverter adds extra converters to the ValueTo conversion.
// See WithValueToConverterFor for more convenience.
func WithValueToConverter(converters ...ValueToConverter) ConverterOption {
	return func(c *converter) {
		c.ValueToConverters = append(c.ValueToConverters, converters...)
	}
}

// WithValueToConverterFor targets the given type t and converts values with the provided ValueToFunc.
func WithValueToConverterFor[T any](f ValueToFunc[T]) ConverterOption {
	t := reflect.TypeFor[T]()
	return WithValueToConverter(func(attributePath path.Path, in tftypes.Value, out reflect.Value) (matched bool, err error) {
		if !out.Type().AssignableTo(t) {
			return // matched=false
		}
		matched = true
		if v, converterErr := f(attributePath, in); converterErr != nil {
			err = converterErr
		} else {
			out.Set(reflect.ValueOf(v))
		}
		return
	})
}

type unknowableAddr interface {
	UnwrapAddr() reflect.Value
}

func (conv converter) valueTo(in tftypes.Value, out reflect.Value, path reflectwalk.WalkPath) (err error) {
	defer func() {
		err = path.WrapError(err)
	}()

	// Run value converters on null input values to allow setting computed, optional values
	if len(conv.ValueToConverters) > 0 {
		attributePath := conv.walkPathToAttributePath(path)
		for _, toConverter := range conv.ValueToConverters {
			if matched, err := toConverter(attributePath, in, out); err != nil {
				return fmt.Errorf("converter %T: %w", toConverter, err)
			} else if matched {
				return nil
			}
		}
	}

	if out.CanAddr() {
		// unwrap UnknownIsNull wrapping struct
		if u, ok := out.Addr().Interface().(unknowableAddr); ok {
			return conv.valueTo(in, u.UnwrapAddr(), path)
		}
	}

	// all further kinds below have some nil-ability,
	// so simply leave them "nil/zero" if in.IsNull() (or unknown)
	if in.IsNull() {
		return nil
	} else if !in.IsKnown() && conv.SetUnknownValueToZero {
		return nil
	}

	switch kind := out.Kind(); kind {
	case reflect.Ptr:
		if out.IsNil() {
			out.Set(reflect.New(out.Type().Elem()))
		}
		return path.WalkPointer(out, func(path reflectwalk.WalkPath, out reflect.Value) error {
			if err := conv.valueTo(in, out, path); err != nil {
				return err
			}
			return path.Stop()
		})
	case reflect.Slice:
		var values []tftypes.Value
		if err := in.As(&values); err != nil {
			return fmt.Errorf("failed converting %s into slice: %w", in.Type(), err)
		}
		out.Set(reflect.MakeSlice(out.Type(), len(values), len(values)))
		if err := path.WalkSlice(out, func(path reflectwalk.WalkPath, idx *reflectwalk.SliceIndex, sliceValue reflect.Value) error {
			if err := conv.valueTo(values[*idx], sliceValue, path); err != nil {
				return err
			}
			return path.Stop()
		}); err != nil {
			return err
		}
	case reflect.Map:
		var values map[string]tftypes.Value
		if err := in.As(&values); err != nil {
			return fmt.Errorf("failed converting %s into map: %w", in.Type(), err)
		}
		out.Set(reflect.MakeMap(out.Type()))
		// initialize map with zero values to make WalkMap iterate it properly (zero values will be overridden later)
		for key := range values {
			out.SetMapIndex(reflect.ValueOf(key), reflect.Zero(out.Type().Elem()))
		}
		return path.WalkMap(out, func(path reflectwalk.WalkPath, mapKey *reflectwalk.MapKey, mapValue reflect.Value) error {
			// provide a pointer target and set it finally
			mapValuePtr := reflect.New(mapValue.Type())
			if err := conv.valueTo(values[mapKey.Name()], mapValuePtr, path); err != nil {
				return err
			}
			out.SetMapIndex(mapKey.Value, mapValuePtr.Elem())
			return path.Stop()
		})
	case reflect.Struct:
		var values map[string]tftypes.Value
		if err := in.As(&values); err != nil {
			return fmt.Errorf("failed converting %s into struct %T: %w", in.Type(), out.Interface(), err)
		}
		if len(values) == 0 {
			return nil
		}
		return path.WalkStruct(out, func(path reflectwalk.WalkPath, field *reflectwalk.StructField, structValue reflect.Value) (err error) {
			tfsdkTag := field.Tag.Get("tfsdk")
			if strings.TrimSpace(tfsdkTag) == "" {
				return fmt.Errorf("tfsdk tag is required on struct %T, field %s", out.Interface(), field.Name)
			} else if tfsdkTag == "-" {
				return path.Stop()
			}
			if attrValue, ok := values[tfsdkTag]; ok {
				if err := conv.valueTo(attrValue, structValue, path); err != nil {
					return err
				}
				return path.Stop()
			} else {
				return fmt.Errorf("could not find attribute %s, struct %T, field %s, available attributes %s",
					tfsdkTag, out.Interface(), field.Name, slices.Collect(maps.Keys(values)))
			}
		})
	default:
		panic(fmt.Sprintf("kind %s not supported", kind))
	}
	return
}

type ValueToFunc[T any] func(attributePath path.Path, in tftypes.Value) (T, error)

// A ValueToConverter gets the out target passed in to investigate the target kind/type.
// An implementation is expected to call [reflect.Value.Set] on the given out if matched is returned as true.
type ValueToConverter func(attributePath path.Path, in tftypes.Value, out reflect.Value) (matched bool, err error)

type valueToConverterMatchingKind[T any] struct {
	ValueToFunc[T]
	UnknownToZero bool
}

func (c valueToConverterMatchingKind[T]) Adapt() ValueToConverter {
	kind := reflect.TypeFor[T]().Kind()
	return func(attributePath path.Path, in tftypes.Value, out reflect.Value) (matched bool, err error) {
		if out.Kind() == kind {
			matched = true
			if c.UnknownToZero && !in.IsKnown() {
				out.Set(reflect.Zero(out.Type()))
				return
			}
			v, err := c.ValueToFunc(attributePath, in)
			if err != nil {
				return matched, err
			}
			out.Set(reflect.ValueOf(v).Convert(out.Type()))
		}
		return
	}
}

func (c valueToConverterMatchingKind[T]) ConvertTo(attributePath path.Path, in tftypes.Value) (any, error) {
	return c.ValueToFunc(attributePath, in)
}

func convertValueTo[T any](_ path.Path, in tftypes.Value) (out T, err error) {
	err = in.As(&out)
	return
}

func convertValueToInt64(attributePath path.Path, in tftypes.Value) (int64, error) {
	float, err := convertValueTo[big.Float](attributePath, in)
	if err != nil {
		return 0, err
	}
	if out, accuracy := float.Int64(); accuracy != big.Exact {
		return 0, fmt.Errorf("converting value %s (%s) to int64 had non-exact accuracy of %s", in, float.String(), accuracy)
	} else {
		return out, nil
	}
}
