package validators

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/meshcloud/terraform-provider-meshstack/client"
)

var _ validator.Set = SupportedPlatforms{}

type SupportedPlatforms struct{}

func (v SupportedPlatforms) Description(ctx context.Context) string {
	return fmt.Sprintf("Ensures supported_platforms is required and non-empty when target_type is %s", client.MeshBuildingBlockTypeTenantLevel)
}

func (v SupportedPlatforms) MarkdownDescription(ctx context.Context) string {
	return fmt.Sprintf("Ensures supported_platforms is required and non-empty when target_type is %s", client.MeshBuildingBlockTypeTenantLevel)
}

func (v SupportedPlatforms) ValidateSet(ctx context.Context, req validator.SetRequest, resp *validator.SetResponse) {
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
	if targetType.ValueString() != string(client.MeshBuildingBlockTypeTenantLevel) {
		return
	}

	// If target_type is TenantTargetType, supported_platforms must be non-null and non-empty
	if req.ConfigValue.IsNull() || req.ConfigValue.IsUnknown() {
		resp.Diagnostics.Append(diag.NewAttributeErrorDiagnostic(
			req.Path,
			"Invalid Attribute Configuration",
			fmt.Sprintf("Attribute %s is required when target_type is %s", req.Path, client.MeshBuildingBlockTypeTenantLevel),
		))
		return
	}

	elements := req.ConfigValue.Elements()
	if len(elements) == 0 {
		resp.Diagnostics.Append(diag.NewAttributeErrorDiagnostic(
			req.Path,
			"Invalid Attribute Configuration",
			fmt.Sprintf("Attribute %s must be non-empty when target_type is %s", req.Path, client.MeshBuildingBlockTypeTenantLevel),
		))
	}
}
