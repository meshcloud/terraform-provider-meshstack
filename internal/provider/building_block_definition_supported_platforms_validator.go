package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ validator.List = supportedPlatformsValidator{}

type supportedPlatformsValidator struct{}

func (v supportedPlatformsValidator) Description(ctx context.Context) string {
	return fmt.Sprintf("Ensures supported_platforms is required and non-empty when target_type is %s", TenantTargetType)
}

func (v supportedPlatformsValidator) MarkdownDescription(ctx context.Context) string {
	return fmt.Sprintf("Ensures supported_platforms is required and non-empty when target_type is %s", TenantTargetType)
}

func (v supportedPlatformsValidator) ValidateList(ctx context.Context, req validator.ListRequest, resp *validator.ListResponse) {
	// Get the target_type from the parent spec object
	var targetType types.String
	resp.Diagnostics.Append(req.Config.GetAttribute(ctx, path.Root("spec").AtName("target_type"), &targetType)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// If target_type is unknown, we can't validate yet
	if targetType.IsUnknown() {
		return
	}

	// Only validate when target_type is TenantTargetType
	if targetType.ValueString() != TenantTargetType {
		return
	}

	// If target_type is TenantTargetType, supported_platforms must be non-null and non-empty
	if req.ConfigValue.IsNull() || req.ConfigValue.IsUnknown() {
		resp.Diagnostics.Append(diag.NewAttributeErrorDiagnostic(
			req.Path,
			"Invalid Attribute Configuration",
			fmt.Sprintf("Attribute %s is required when target_type is %s", req.Path, TenantTargetType),
		))
		return
	}

	elements := req.ConfigValue.Elements()
	if len(elements) == 0 {
		resp.Diagnostics.Append(diag.NewAttributeErrorDiagnostic(
			req.Path,
			"Invalid Attribute Configuration",
			fmt.Sprintf("Attribute %s must be non-empty when target_type is %s", req.Path, TenantTargetType),
		))
	}
}
