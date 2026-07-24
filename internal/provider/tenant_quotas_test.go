package provider

import (
	"strings"
	"testing"

	clientTypes "github.com/meshcloud/terraform-provider-meshstack/client/types"

	"github.com/meshcloud/terraform-provider-meshstack/client"
)

func TestQuotaRealizationWarning(t *testing.T) {
	tests := map[string]struct {
		requested   map[string]int64
		applied     map[string]int64
		wantWarning bool
		wantDetail  []string // substrings expected in the detail, when a warning is produced
	}{
		"nothing requested": {
			requested:   nil,
			applied:     map[string]int64{"limits.cpu": 4},
			wantWarning: false,
		},
		"applied verbatim": {
			requested:   map[string]int64{"limits.cpu": 4, "limits.memory": 8},
			applied:     map[string]int64{"limits.cpu": 4, "limits.memory": 8},
			wantWarning: false,
		},
		"value differs": {
			requested:   map[string]int64{"limits.cpu": 4000},
			applied:     map[string]int64{"limits.cpu": 2000},
			wantWarning: true,
			wantDetail:  []string{`"limits.cpu": requested 4000, applied 2000`, "platform operator"},
		},
		"not yet applied": {
			requested:   map[string]int64{"limits.cpu": 4},
			applied:     nil,
			wantWarning: true,
			wantDetail:  []string{`"limits.cpu": requested 4, not yet applied`},
		},
		"one of several differs": {
			requested:   map[string]int64{"limits.cpu": 4, "limits.memory": 8},
			applied:     map[string]int64{"limits.cpu": 4, "limits.memory": 6},
			wantWarning: true,
			wantDetail:  []string{`"limits.memory": requested 8, applied 6`},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			summary, detail, ok := quotaRealizationWarning(tc.requested, tc.applied)
			if ok != tc.wantWarning {
				t.Fatalf("quotaRealizationWarning ok = %v, want %v (detail: %q)", ok, tc.wantWarning, detail)
			}
			if !ok {
				return
			}
			if summary == "" {
				t.Error("expected a non-empty summary when a warning is produced")
			}
			for _, want := range tc.wantDetail {
				if !strings.Contains(detail, want) {
					t.Errorf("detail %q does not contain %q", detail, want)
				}
			}
		})
	}
}

func TestRequestedQuotaValues(t *testing.T) {
	t.Run("prefers requested_quotas map", func(t *testing.T) {
		spec := client.MeshTenantSpec{
			RequestedQuotas: map[string]client.RequestQuotaValue{"limits.cpu": {Value: 4}},
			Quotas:          clientTypes.Set[client.MeshTenantQuota]{{Key: "limits.cpu", Value: 99}},
		}
		got := requestedQuotaValues(spec)
		if len(got) != 1 || got["limits.cpu"] != 4 {
			t.Fatalf("got %v, want map[limits.cpu:4]", got)
		}
	})

	t.Run("falls back to deprecated quotas list", func(t *testing.T) {
		spec := client.MeshTenantSpec{
			Quotas: clientTypes.Set[client.MeshTenantQuota]{{Key: "limits.cpu", Value: 7}},
		}
		got := requestedQuotaValues(spec)
		if len(got) != 1 || got["limits.cpu"] != 7 {
			t.Fatalf("got %v, want map[limits.cpu:7]", got)
		}
	})

	t.Run("nil when nothing requested", func(t *testing.T) {
		if got := requestedQuotaValues(client.MeshTenantSpec{}); got != nil {
			t.Fatalf("got %v, want nil", got)
		}
	})
}
