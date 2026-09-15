package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// wifProviderStatusAttribute builds the computed per-cloud block of a resolved workload identity
// federation: the values a building block run presents to that cloud.
func wifProviderStatusAttribute(cloud string) schema.SingleNestedAttribute {
	return schema.SingleNestedAttribute{
		MarkdownDescription: "Workload identity federation values for " + cloud + ". Null when the runner does not federate with " + cloud + ".",
		Computed:            true,
		Attributes: map[string]schema.Attribute{
			"audience": schema.StringAttribute{
				MarkdownDescription: "Audience the federated identity token is issued for.",
				Computed:            true,
			},
			"token_path": schema.StringAttribute{
				MarkdownDescription: "Path the runner writes that token to.",
				Computed:            true,
			},
		},
	}
}

// useStateUnlessStatusChanges is UseStateForUnknown with two exceptions. A computed object attribute is
// planned unknown on every update, so without it an unrelated change would report status as "known after
// apply" - but plain UseStateForUnknown would be wrong whenever status really changes, and carrying the
// stale value over would end the apply with "Provider produced inconsistent result after apply". meshStack
// resolves a version's identity against the runner of that version, so a changed version_spec.runner_ref
// changes the latest entry, and a draft cut from a released version (draft false -> true) adds one. In both
// cases status stays unknown and is read again after the version is written.
type useStateUnlessStatusChanges struct{}

func (m useStateUnlessStatusChanges) Description(context.Context) string {
	return "Keeps the value from state unless `version_spec.runner_ref` changes or a new version is created, in which case it is read again."
}

func (m useStateUnlessStatusChanges) MarkdownDescription(ctx context.Context) string {
	return m.Description(ctx)
}

func (m useStateUnlessStatusChanges) PlanModifyObject(ctx context.Context, req planmodifier.ObjectRequest, resp *planmodifier.ObjectResponse) {
	if !req.PlanValue.IsUnknown() || req.State.Raw.IsNull() || req.StateValue.IsNull() {
		return
	}

	versionSpecPath := path.Root("version_spec")
	var plannedRunner, currentRunner types.Object
	var plannedDraft, currentDraft types.Bool
	resp.Diagnostics.Append(req.Plan.GetAttribute(ctx, versionSpecPath.AtName("runner_ref"), &plannedRunner)...)
	resp.Diagnostics.Append(req.State.GetAttribute(ctx, versionSpecPath.AtName("runner_ref"), &currentRunner)...)
	resp.Diagnostics.Append(req.Plan.GetAttribute(ctx, versionSpecPath.AtName("draft"), &plannedDraft)...)
	resp.Diagnostics.Append(req.State.GetAttribute(ctx, versionSpecPath.AtName("draft"), &currentDraft)...)
	if resp.Diagnostics.HasError() || !plannedRunner.Equal(currentRunner) {
		return
	}
	newVersion := plannedDraft.IsUnknown() || (plannedDraft.ValueBool() && !currentDraft.ValueBool())
	if newVersion {
		return
	}

	resp.PlanValue = req.StateValue
}
