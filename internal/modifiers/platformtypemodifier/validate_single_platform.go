package platformtypemodifier

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
)

// ValidateSinglePlatform returns a plan modifier that validates that only one platform
// configuration is specified at a time.
func ValidateSinglePlatform() planmodifier.Object {
	return singlePlatformValidator{}
}

type singlePlatformValidator struct{}

// Description returns a human-readable description of the plan modifier.
func (v singlePlatformValidator) Description(_ context.Context) string {
	return "Validates that only one platform configuration is specified"
}

// MarkdownDescription returns a markdown description of the plan modifier.
func (v singlePlatformValidator) MarkdownDescription(_ context.Context) string {
	return "Validates that only one platform configuration is specified"
}

// PlanModifyObject implements the plan modification logic.
func (v singlePlatformValidator) PlanModifyObject(ctx context.Context, req planmodifier.ObjectRequest, resp *planmodifier.ObjectResponse) {
	if req.PlanValue.IsNull() || req.PlanValue.IsUnknown() {
		return
	}

	platformTypes := []string{"aws", "aks", "azure", "azurerg", "gcp", "kubernetes", "openshift"}

	var configuredPlatforms []string
	for _, platformType := range platformTypes {
		attrValue := req.PlanValue.Attributes()[platformType]
		if attrValue != nil && !attrValue.IsNull() {
			configuredPlatforms = append(configuredPlatforms, platformType)
		}
	}

	if len(configuredPlatforms) > 1 {
		resp.Diagnostics.AddError(
			"Multiple Platform Configurations",
			fmt.Sprintf("Only one platform configuration can be specified,"+
				"but found: %v. Please specify only one platform "+
				"configuration.", configuredPlatforms),
		)
	}

	if len(configuredPlatforms) == 0 {
		resp.Diagnostics.AddError(
			"No Platform Configuration",
			"At least one platform configuration must be specified. "+
				"Please specify one of: aws, aks, azure, azurerg, gcp, kubernetes, openshift.",
		)
	}
}
