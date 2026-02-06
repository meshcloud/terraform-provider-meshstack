package generic

import (
	"context"
	"encoding/json"
	"fmt"
	"math/big"

	"github.com/hashicorp/terraform-plugin-framework-jsontypes/jsontypes"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/attr/xattr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

type Value[T any] struct {
	attr.Value
	attributePath *path.Path
}

var (
	// Ensure the implementation satisfies the expected interfaces to the Terraform Framework.
	// Must match the supported types in TypeFor.
	// Note that T=any is represented as jsontypes.Normalized, which is a basetypes.StringValuableWithSemanticEquals.
	_ basetypes.StringValuable                   = Value[any]{}
	_ basetypes.StringValuableWithSemanticEquals = Value[any]{}
	_ basetypes.BoolValuable                     = Value[any]{}
	_ basetypes.Int64Valuable                    = Value[any]{}
	_ basetypes.ObjectValuable                   = Value[any]{}
	// Implementing this allows us to catch and set the Value.attributePath field!
	_ xattr.ValidateableAttribute = Value[any]{}
)

func (v *Value[T]) SetRequired(in *T, diags *diag.Diagnostics) {
	if in == nil {
		v.addAttributeErrorf(diags, "Required input not present", "The value of type %T is required, but nil input was provided.", *v)
		return
	}
	v.SetOptional(in, diags)
}

func (v Value[T]) addAttributeErrorf(diags *diag.Diagnostics, summary, format string, args ...any) {
	details := fmt.Sprintf(format, args...)
	if v.attributePath != nil {
		diags.AddAttributeError(*v.attributePath, summary, details)
	} else {
		diags.AddError(summary, details)
	}
}

func (v *Value[T]) SetOptional(in *T, diags *diag.Diagnostics) {
	if in == nil {
		*v = v.null()
		return
	}
	*v = TypeFor[T]().valueFactory(*in, diags)
}

func (v Value[T]) Get(diags *diag.Diagnostics) (result T) {
	if v.IsUnknown() {
		v.addAttributeErrorf(diags, "generic.Value.Get failed", "Getting an unknown generic value is impossible")
		return
	}
	if jsonValue, ok := v.Value.(jsontypes.Normalized); ok {
		if v.IsNull() {
			// null must be handled explicitly as Unmarshal complains otherwise
			// keeping the result untouched will use the default zero value for T,
			// which is nil in case of T=any
			return
		}
		diags.Append(jsonValue.Unmarshal(&result)...)
		return
	} else {
		if v.IsNull() {
			v.addAttributeErrorf(diags, "generic.Value.Get failed", "Getting a null generic value as non-pointer primitive (non-any) is impossible. Use generic.Value.GetPtr instead.")
			return
		}
		return v.asPrimitive(diags)
	}
}

func (v Value[T]) GetPtr(diags *diag.Diagnostics) (result *T) {
	switch {
	case v.IsNull() || v.IsUnknown():
		// just leave result nil, even when unknown which is handy when mapping model to JSON representation in resources
		return
	default:
		if _, ok := v.Value.(jsontypes.Normalized); ok {
			v.addAttributeErrorf(diags, "generic.Value.GetPtr failed", "Getting a pointer to value of type %s (any) is not allowed", Type[T]{})
			return
		} else {
			valueResult := v.asPrimitive(diags)
			return &valueResult
		}
	}
}

func (v Value[T]) asPrimitive(diags *diag.Diagnostics) (result T) {
	if objectValue, ok := v.Value.(basetypes.ObjectValue); ok {
		diags.Append(objectValue.As(context.Background(), &result, basetypes.ObjectAsOptions{})...)
		return
	}
	tfValue, err := v.ToTerraformValue(context.Background())
	if err != nil {
		v.addAttributeErrorf(diags, "Error in generic.Value.asPrimitive", "Failed to convert generic value of type %T to Terraform value: %s", v, err.Error())
		return
	}
	var target any
	if tfValue.Type().Is(tftypes.Number) {
		target = &big.Float{}
	} else {
		target = &result
	}
	if err := tfValue.As(target); err != nil {
		v.addAttributeErrorf(diags, "Error in generic.Value.asPrimitive", "Failed to convert Terraform value of type %s to result of type %T: %s", tfValue.Type(), result, err.Error())
	}
	if float, ok := target.(*big.Float); ok {
		mapper := TypeFor[T]().bigFloatMapper
		if mapper == nil {
			v.addAttributeErrorf(diags, "Error in generic.Value.asPrimitive", "No mapper for big.Float for generic type %T", v)
			return
		}
		return mapper(float, diags)
	}
	return
}

func bigFloatMapper[T, Number any](mapper func(*big.Float) (Number, big.Accuracy)) func(*big.Float, *diag.Diagnostics) T {
	return func(float *big.Float, diags *diag.Diagnostics) (result T) {
		number, accuracy := mapper(float)
		if accuracy != big.Exact {
			diags.AddError("Number conversion error", fmt.Sprintf("The Terraform number type %T (from big.Float) cannot be converted exactly to target type %T", number, result))
		}
		var ok bool
		if result, ok = any(number).(T); !ok {
			diags.AddError("Number conversion error", fmt.Sprintf("The Terraform number type %T (from big.Float) cannot be cast to target type %T", number, result))
		}
		return
	}
}

func (v Value[T]) null() Value[T] {
	return newValue[T](TypeFor[T]().underlyingNull)
}

func newValue[T any](attrValue attr.Value) Value[T] {
	return Value[T]{attrValue, &path.Path{}}
}

func anyValueFactory[T any](v any, diags *diag.Diagnostics) (result Value[T]) {
	if vPtr, ok := v.(*any); ok {
		if vPtr == nil {
			return result.null()
		} else {
			// deference "pointer-to-any" to simply any for proper marshaling below
			v = *vPtr
		}
	}
	if v == nil {
		return result.null()
	}
	data, err := json.Marshal(v)
	if err == nil {
		return newValue[T](jsontypes.NewNormalizedValue(string(data)))
	} else {
		diags.AddError("Marshaling Value[any] failed", err.Error())
		return result.null()
	}
}

func (v Value[T]) Type(_ context.Context) attr.Type {
	return TypeFor[T]()
}

func (v Value[T]) Equal(value attr.Value) bool {
	if genericValue, ok := value.(Value[T]); ok {
		return v.Value.Equal(genericValue.Value)
	} else {
		return false
	}
}

func (v Value[T]) StringSemanticEquals(ctx context.Context, valuable basetypes.StringValuable) (bool, diag.Diagnostics) {
	if withSemanticEquals, ok := v.Value.(basetypes.StringValuableWithSemanticEquals); ok {
		return withSemanticEquals.StringSemanticEquals(ctx, valuable)
	} else {
		return v.Equal(valuable), nil
	}
}

func (v Value[T]) ValidateAttribute(ctx context.Context, request xattr.ValidateAttributeRequest, response *xattr.ValidateAttributeResponse) {
	if v.attributePath != nil {
		*v.attributePath = request.Path.Copy()
	}
	if v.Value == nil {
		response.Diagnostics.AddAttributeError(request.Path, "generic.Value.ValidateAttribute failed", "Value is nil")
	} else if validatable, ok := v.Value.(xattr.ValidateableAttribute); ok {
		validatable.ValidateAttribute(ctx, request, response)
	}
}

func toValue[T, V, R any](v Value[T], ctx context.Context, mapper func(V, context.Context) (R, diag.Diagnostics)) (result R, diags diag.Diagnostics) {
	if valuable, ok := v.Value.(V); ok {
		return mapper(valuable, ctx)
	} else {
		v.addAttributeErrorf(&diags, "Generic Value Conversion Error", "Attribute type %s not a %T", v.Type(ctx), valuable)
	}
	return
}

func (v Value[T]) ToStringValue(ctx context.Context) (result basetypes.StringValue, diags diag.Diagnostics) {
	return toValue(v, ctx, basetypes.StringValuable.ToStringValue)
}

func (v Value[T]) ToBoolValue(ctx context.Context) (basetypes.BoolValue, diag.Diagnostics) {
	return toValue(v, ctx, basetypes.BoolValuable.ToBoolValue)
}

func (v Value[T]) ToInt64Value(ctx context.Context) (result basetypes.Int64Value, diags diag.Diagnostics) {
	return toValue(v, ctx, basetypes.Int64Valuable.ToInt64Value)
}

func (v Value[T]) ToObjectValue(ctx context.Context) (basetypes.ObjectValue, diag.Diagnostics) {
	return toValue(v, ctx, basetypes.ObjectValuable.ToObjectValue)
}
