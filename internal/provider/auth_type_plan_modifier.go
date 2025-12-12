package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// authTypeModifier sets the auth type based on which credential field is populated
type authTypeModifier struct{}

func (m authTypeModifier) Description(ctx context.Context) string {
	return "Sets auth type to 'credential' if credential is set, 'workloadIdentity' otherwise"
}

func (m authTypeModifier) MarkdownDescription(ctx context.Context) string {
	return "Sets auth type to 'credential' if credential is set, 'workloadIdentity' otherwise"
}

func (m authTypeModifier) PlanModifyString(ctx context.Context, req planmodifier.StringRequest, resp *planmodifier.StringResponse) {
	// Get the parent object (auth config)
	var authConfig types.Object
	resp.Diagnostics.Append(req.Plan.GetAttribute(ctx, req.Path.ParentPath(), &authConfig)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if authConfig.IsNull() || authConfig.IsUnknown() {
		return
	}

	// Extract the credential field from the auth config
	attrs := authConfig.Attributes()
	credentialAttr, exists := attrs["credential"]
	if !exists {
		return
	}

	// Check if credential is set (not null)
	credentialObj, ok := credentialAttr.(types.Object)
	if !ok {
		return
	}

	// Set type based on whether credential is populated
	if credentialObj.IsNull() || credentialObj.IsUnknown() {
		resp.PlanValue = types.StringValue("workloadIdentity")
	} else {
		resp.PlanValue = types.StringValue("credential")
	}
}

func authTypeDefault() planmodifier.String {
	return authTypeModifier{}
}

var _ planmodifier.String = authTypeModifier{}
