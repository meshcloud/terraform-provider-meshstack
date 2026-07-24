package provider

import (
	"fmt"
	"sort"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/diag"

	"github.com/meshcloud/terraform-provider-meshstack/client"
)

// requestedQuotaValues flattens the quotas a caller requested — from the preferred requested_quotas
// map or, failing that, the deprecated quotas list — into a plain key->value map. Returns nil when no
// quotas were requested.
func requestedQuotaValues(spec client.MeshTenantSpec) map[string]int64 {
	if len(spec.RequestedQuotas) > 0 {
		out := make(map[string]int64, len(spec.RequestedQuotas))
		for k, v := range spec.RequestedQuotas {
			out[k] = v.Value
		}
		return out
	}
	if len(spec.Quotas) > 0 {
		out := make(map[string]int64, len(spec.Quotas))
		for _, q := range spec.Quotas {
			out[q.Key] = q.Value
		}
		return out
	}
	return nil
}

// appliedQuotaValues flattens status.applied_quotas into a plain key->value map. Returns nil when empty.
func appliedQuotaValues(status client.MeshTenantStatus) map[string]int64 {
	if len(status.AppliedQuotas) == 0 {
		return nil
	}
	out := make(map[string]int64, len(status.AppliedQuotas))
	for k, v := range status.AppliedQuotas {
		out[k] = v.Value
	}
	return out
}

// quotaRealizationWarning compares the quotas a caller requested against the quotas meshStack actually
// applied and returns a warning (summary, detail, ok=true) when a requested quota was not realized to
// the requested value. A mismatch is expected rather than an error: meshStack merges the landing zone's
// default quotas with the request, and a requested increase beyond a platform's auto-approval threshold
// needs a platform operator to approve the quota request before it takes effect, so the applied value
// can differ from — or lag behind — what was requested. Returns ok=false when nothing was requested or
// every requested quota was applied verbatim.
func quotaRealizationWarning(requested, applied map[string]int64) (summary, detail string, ok bool) {
	if len(requested) == 0 {
		return "", "", false
	}

	keys := make([]string, 0, len(requested))
	for k := range requested {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	var lines []string
	for _, k := range keys {
		want := requested[k]
		got, present := applied[k]
		switch {
		case !present:
			lines = append(lines, fmt.Sprintf("- %q: requested %d, not yet applied", k, want))
		case got != want:
			lines = append(lines, fmt.Sprintf("- %q: requested %d, applied %d", k, want, got))
		}
	}
	if len(lines) == 0 {
		return "", "", false
	}

	summary = "Requested tenant quotas were not fully applied"
	detail = "meshStack applied quota values that differ from what was requested:\n" +
		strings.Join(lines, "\n") +
		"\n\nThis is usually expected: the landing zone's default quotas are merged with the request, and a " +
		"requested increase beyond a platform's auto-approval threshold needs a platform operator to approve the " +
		"quota request before it takes effect. Review the tenant's quota requests in the meshStack panel " +
		"(Tenant > Settings > Quotas) if a value should already have been applied."
	return summary, detail, true
}

// warnOnUnrealizedQuotas appends a quota-realization warning to diags when the tenant's requested
// quotas were not applied verbatim.
func warnOnUnrealizedQuotas(spec client.MeshTenantSpec, status client.MeshTenantStatus, diags *diag.Diagnostics) {
	if summary, detail, ok := quotaRealizationWarning(requestedQuotaValues(spec), appliedQuotaValues(status)); ok {
		diags.AddWarning(summary, detail)
	}
}
