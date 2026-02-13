package generic

import (
	"context"
	"fmt"
	"reflect"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

// AttributeSetter are implemented by Terraform State, Plan, Config.
type AttributeSetter interface {
	Set(context.Context, any) diag.Diagnostics
	SetAttribute(context.Context, path.Path, any) diag.Diagnostics
}

// Set sets the whole Terraform value converted from the given input using ValueFrom.
func Set[T any](ctx context.Context, attributeSetter AttributeSetter, in T, opts ...ConverterOption) diag.Diagnostics {
	return set(in, opts, func(value tftypes.Value) diag.Diagnostics {
		return attributeSetter.Set(ctx, value)
	})
}

// SetPartial only sets the attribute parts present in the input T.
// Useful when modifying/setting plans only partially (targeting computed values only).
func SetPartial[T any](ctx context.Context, attributeSetter AttributeSetter, in T) diag.Diagnostics {
	var diags diag.Diagnostics
	opts := ConverterOptions{
		WithValueFromConverter(func(attributePath path.Path, in reflect.Value, haveNil, haveUnknown bool) (out tftypes.Value, matched bool, err error) {
			switch kind := in.Kind(); kind {
			case reflect.Slice, reflect.Map, reflect.Struct:
				// continue traversing (matched=false) only if not nil
				if haveNil {
					matched = true
				}
			case reflect.Bool, reflect.String, reflect.Int64:
				matched = true
			default:
				panic(fmt.Errorf("unsupported kind: %s", kind))
			}
			if matched {
				if haveNil {
					in = reflect.Zero(reflect.New(in.Type()).Type())
				}
				out, err = ValueFrom(in.Interface())
				if haveUnknown {
					out = tftypes.NewValue(out.Type(), tftypes.UnknownValue)
				}
				diags.Append(attributeSetter.SetAttribute(ctx, attributePath, out)...)
			}
			return
		}),
	}
	return set(in, opts, func(value tftypes.Value) diag.Diagnostics {
		return diags // do nothing, as partial setting happens already above
	})
}

// SetAttributeTo sets the whole attribute to the given input, converted with ValueFrom.
func SetAttributeTo[T any](ctx context.Context, attributeSetter AttributeSetter, attributePath path.Path, in T, opts ...ConverterOption) diag.Diagnostics {
	return set(in, ConverterOptions{WithAttributePath(attributePath)}.Append(opts...), func(tfValue tftypes.Value) diag.Diagnostics {
		return attributeSetter.SetAttribute(ctx, attributePath, tfValue)
	})
}

func set[T any](in T, opts []ConverterOption, setter func(tftypes.Value) diag.Diagnostics) (diags diag.Diagnostics) {
	tfValue, err := ValueFrom(in, opts...)
	if err != nil {
		diags.AddError(fmt.Sprintf("Converting from generic type %T failed", in), err.Error())
		return
	}
	return setter(tfValue)
}
