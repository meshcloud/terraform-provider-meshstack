package platformtypemodifier

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// SetTypeFromPlatform returns a plan modifier that automatically sets the type field
// based on which platform-specific configuration is present.
func SetTypeFromPlatform() planmodifier.String {
	return platformTypeModifier{}
}

type platformTypeModifier struct{}

// Description returns a human-readable description of the plan modifier.
func (m platformTypeModifier) Description(_ context.Context) string {
	return "Automatically sets type based on which platform configuration is provided"
}

// MarkdownDescription returns a markdown description of the plan modifier.
func (m platformTypeModifier) MarkdownDescription(_ context.Context) string {
	return "Automatically sets `type` based on which platform configuration is provided"
}

// PlanModifyString implements the plan modification logic.
func (m platformTypeModifier) PlanModifyString(ctx context.Context, req planmodifier.StringRequest, resp *planmodifier.StringResponse) {
	// Get the parent object (platform_properties)
	platformPropsPath := req.Path.ParentPath()
	var platformProps types.Object
	resp.Diagnostics.Append(req.Plan.GetAttribute(ctx, platformPropsPath, &platformProps)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if platformProps.IsNull() || platformProps.IsUnknown() {
		return
	}

	// Check which platform-specific configuration is present
	platformTypes := []string{"aws", "aks", "azure", "azurerg", "gcp", "kubernetes", "openshift"}

	for _, platformType := range platformTypes {
		attrValue := platformProps.Attributes()[platformType]
		if attrValue != nil && !attrValue.IsNull() {
			resp.PlanValue = types.StringValue(platformType)
			// Early return is okay here: if more than one platform-property is set, the singlePlatformValidator
			// will catch this.
			return
		}
	}
}
