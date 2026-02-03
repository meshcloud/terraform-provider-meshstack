package generic

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
)

func FromObject[T any](ctx context.Context, obj basetypes.ObjectValue, diags *diag.Diagnostics) (result T) {
	diags.Append(obj.As(ctx, &result, basetypes.ObjectAsOptions{})...)
	return
}

func ToObject[T any](ctx context.Context, in T, attrTypesOf func(context.Context) map[string]attr.Type, diagsOut *diag.Diagnostics) basetypes.ObjectValue {
	result, diags := basetypes.NewObjectValueFrom(ctx, attrTypesOf(ctx), in)
	diagsOut.Append(diags...)
	return result
}

func SetAttribute(ctx context.Context, target *basetypes.ObjectValue, key string, val attr.Value, diagsOut *diag.Diagnostics) {
	attrs := target.Attributes()
	attrs[key] = val
	var diags diag.Diagnostics
	*target, diags = basetypes.NewObjectValue(target.AttributeTypes(ctx), attrs)
	diagsOut.Append(diags...)
}

func AppendElement(ctx context.Context, target *basetypes.ListValue, elem attr.Value, diagsOut *diag.Diagnostics) {
	withElements(ctx, target, diagsOut, func(elements []attr.Value) []attr.Value {
		return append(elements, elem)
	})
}

func SetLastElement(ctx context.Context, target *basetypes.ListValue, elem attr.Value, diagsOut *diag.Diagnostics) {
	withElements(ctx, target, diagsOut, func(elements []attr.Value) []attr.Value {
		elements[len(elements)-1] = elem
		return elements
	})
}

func withElements(ctx context.Context, target *basetypes.ListValue, diagsOut *diag.Diagnostics, withElements func(elements []attr.Value) []attr.Value) {
	elems := withElements(target.Elements())
	var diags diag.Diagnostics
	*target, diags = basetypes.NewListValue(target.ElementType(ctx), elems)
	diagsOut.Append(diags...)
}
