package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// reconcileTrackedTags restricts apiTags to the tag keys already tracked in state at tagsPath. The
// meshObject API returns a superset of the tags a caller sent — an entry for every schema property
// (empty list for unset ones) plus injected restricted-tag defaults — which the caller may be unable
// to manage. Keeping only the previously tracked keys prevents those server-side additions from
// entering the user-managed `tags` attribute and producing spurious drift on the next plan.
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
// tracked, dropping the server-injected superset entries (empty lists for undeclared properties and
// restricted-tag defaults) that are not tracked in state.
func reconcileTags(tracked, apiTags map[string][]string) map[string][]string {
	reconciled := make(map[string][]string, len(tracked))
	for key := range tracked {
		if value, ok := apiTags[key]; ok {
			reconciled[key] = value
		}
	}
	return reconciled
}
