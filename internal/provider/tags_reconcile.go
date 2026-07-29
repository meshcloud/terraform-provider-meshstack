package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// reconcileTrackedTags restricts apiTags to the tag keys already tracked in state at tagsPath. The
// meshObject API can return tags the caller never sent — chiefly the restricted-tag defaults meshStack
// injects for projects, landing zones, building block definitions and panel-registered workspaces —
// which the caller may be unable to manage. Keeping only the previously tracked keys prevents those
// server-side additions from entering the user-managed `tags` attribute and producing spurious drift on
// the next plan.
//
// On import there is no prior state (tags is null), so apiTags is returned unchanged and the full set
// round-trips. Reading state can fail, so diagnostics are appended to diags; check diags.HasError()
// at the call site as usual.
func reconcileTrackedTags(ctx context.Context, state tfsdk.State, tagsPath path.Path, apiTags map[string][]string, diags *diag.Diagnostics) map[string][]string {
	var priorTags types.Map
	diags.Append(state.GetAttribute(ctx, tagsPath, &priorTags)...)
	if diags.HasError() || priorTags.IsNull() {
		return apiTags
	}

	var tracked map[string][]string
	diags.Append(priorTags.ElementsAs(ctx, &tracked, false)...)
	if diags.HasError() {
		return apiTags
	}

	return reconcileTags(tracked, apiTags)
}

// reconcileTags is the pure core of reconcileTrackedTags: it restricts apiTags to the keys present in
// tracked, dropping server-side entries (such as injected restricted-tag defaults) that are not tracked
// in state.
//
// A tracked key the API does not return is treated as deleted — except when its tracked value list is
// empty. The API cannot represent a tag with no values: a meshWorkspace PUT carrying `{"k": []}` comes
// back with `"tags": {}`, so "absent" and "declared empty" are the same state on the wire. Dropping
// such a key would leave a plan that re-adds it on every apply and never converges.
func reconcileTags(tracked, apiTags map[string][]string) map[string][]string {
	reconciled := make(map[string][]string, len(tracked))
	for key, trackedValue := range tracked {
		switch value, ok := apiTags[key]; {
		case ok:
			reconciled[key] = value
		case len(trackedValue) == 0:
			reconciled[key] = []string{}
		}
	}
	return reconciled
}
