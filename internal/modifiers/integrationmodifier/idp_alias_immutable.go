package integrationmodifier

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
)

// idpAliasImmutableModifier rejects a change to an Entra ID integration's identity provider alias.
//
// meshStack cannot repoint an integration at another identity provider, so the only way to express the
// change would be a destroy and recreate. Deleting the integration deletes the identity provider that
// carries the alias, which for an adopted alias is one the customer configured themselves — too much to
// do for an edited string. Reject the change instead, the way version_spec does in
// building_block_definition_resource.go. The framework ships no equivalent: stringplanmodifier offers
// only RequiresReplace variants and UseStateForUnknown, and a validator cannot see prior state.
type idpAliasImmutableModifier struct{}

func (m idpAliasImmutableModifier) Description(ctx context.Context) string {
	return "Rejects a change to the identity provider alias"
}

func (m idpAliasImmutableModifier) MarkdownDescription(ctx context.Context) string {
	return m.Description(ctx)
}

func (m idpAliasImmutableModifier) PlanModifyString(ctx context.Context, req planmodifier.StringRequest, resp *planmodifier.StringResponse) {
	// Not applicable on create (no prior state) or destroy (no plan), nor when the configuration leaves
	// the alias to meshStack.
	if req.State.Raw.IsNull() || req.Plan.Raw.IsNull() || req.ConfigValue.IsNull() || req.ConfigValue.IsUnknown() {
		return
	}
	if req.ConfigValue.Equal(req.StateValue) {
		return
	}

	// State written before the alias existed, or a plan run with -refresh=false, leaves it unknown to
	// Terraform. Refuse either way: letting the request go out only moves the failure past the apply.
	if req.StateValue.IsNull() {
		resp.Diagnostics.AddAttributeError(req.Path, "Error updating idp_alias", fmt.Sprintf(
			"The identity provider alias of an Entra ID integration cannot be changed, and Terraform does "+
				"not know the current one, so it cannot tell whether %q is a change. Refresh the state and "+
				"try again.",
			req.ConfigValue.ValueString(),
		))
		return
	}

	resp.Diagnostics.AddAttributeError(req.Path, "Error updating idp_alias", fmt.Sprintf(
		"The identity provider alias of an Entra ID integration cannot be changed. It is %q and the "+
			"configuration asks for %q. Applying that would delete identity provider %q, which meshStack "+
			"may not have created. Restore the current value, or remove this resource from state and "+
			"import the integration that already uses %q.",
		req.StateValue.ValueString(), req.ConfigValue.ValueString(),
		req.StateValue.ValueString(), req.ConfigValue.ValueString(),
	))
}

// IdpAliasImmutable rejects a change to an Entra ID integration's identity provider alias.
func IdpAliasImmutable() planmodifier.String {
	return idpAliasImmutableModifier{}
}

var _ planmodifier.String = idpAliasImmutableModifier{}
