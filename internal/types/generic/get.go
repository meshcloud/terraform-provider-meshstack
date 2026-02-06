package generic

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
)

// AttributeGetter are implemented by Terraform State, Plan, Config.
type AttributeGetter interface {
	Get(ctx context.Context, target any) diag.Diagnostics
	GetAttribute(context.Context, path.Path, any) diag.Diagnostics
}

// Get gets the whole given Terraform value and converts to T using ValueTo.
func Get[T any](ctx context.Context, attributeGetter AttributeGetter, diags *diag.Diagnostics, opts ...ConverterOption) (out T) {
	return GetAttribute[T](ctx, attributeGetter, path.Empty(), diags, opts...)
}

// GetAttribute gets a sub-part according to the attribute path.
// See also Get.
func GetAttribute[T any](ctx context.Context, attributeGetter AttributeGetter, attributePath path.Path, diags *diag.Diagnostics, opts ...ConverterOption) (out T) {
	var attributeValue attr.Value
	diags.Append(attributeGetter.GetAttribute(ctx, attributePath, &attributeValue)...)
	if diags.HasError() {
		return
	}
	in, err := attributeValue.ToTerraformValue(ctx)
	if err != nil {
		diags.AddError(fmt.Sprintf("Converting to Terraform value for generic type %T failed", out), err.Error())
	}
	out, err = ValueTo[T](in, append([]ConverterOption{WithAttributePath(attributePath)}, opts...)...)
	if err != nil {
		diags.AddError(fmt.Sprintf("Converting to generic type %T failed, got from %T", out, attributeGetter), err.Error())
	}
	return out
}
