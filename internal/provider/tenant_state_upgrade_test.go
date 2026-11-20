package provider

import (
	_ "embed"
	"testing"
)

// As provider v0.24.3 wrote it.
var (
	//go:embed testdata/tenant/state-v1-deprecated-quotas.json
	tenantStateV1DeprecatedQuotas string
	//go:embed testdata/tenant/state-v1-requested-quotas.json
	tenantStateV1RequestedQuotas string
	//go:embed testdata/tenant/state-v1-no-quotas.json
	tenantStateV1NoQuotas string
)

func TestTenantUpgradeState(t *testing.T) {
	t.Run("v1_to_v2", func(t *testing.T) {
		if !HasNestedAttributeForTest(t, tenantSchemaV1Once(), "spec", "quotas") {
			t.Error("prior v1 schema lost spec.quotas; state written by v0.24.3 carries that key")
		}
		if HasNestedAttributeForTest(t, ResourceSchemaForTest(t, &tenantResource{}), "spec", "quotas") {
			t.Fatal("current schema still declares spec.quotas")
		}

		t.Run("translates the deprecated list into requested_quotas", func(t *testing.T) {
			var upgraded tenantResourceModel
			diags := UpgradeResourceStateFromJSON(t, &tenantResource{}, 1, tenantStateV1DeprecatedQuotas, &upgraded)
			if diags.HasError() {
				t.Fatalf("upgrade produced errors: %s", diags)
			}

			want := map[string]int64{"limits.cpu": 4, "limits.memory": 8}
			if len(upgraded.Spec.RequestedQuotas) != len(want) {
				t.Fatalf("requested_quotas not translated, got %v", upgraded.Spec.RequestedQuotas)
			}
			for k, v := range want {
				if got := upgraded.Spec.RequestedQuotas[k]; got.Value != v {
					t.Errorf("requested_quotas[%q] = %d, want %d", k, got.Value, v)
				}
			}

			if *upgraded.Metadata.Uuid != "8f1c2d3e-4a5b-6c7d-8e9f-0a1b2c3d4e5f" {
				t.Errorf("metadata not carried over, got %+v", upgraded.Metadata)
			}
			if upgraded.Spec.PlatformRef.Uuid != "a2b7cf9c-8e2a-4b0d-9e6f-1c4b9de3a111" {
				t.Errorf("platform_ref not carried over, got %+v", upgraded.Spec.PlatformRef)
			}
			if upgraded.Spec.LandingZoneRef == nil || upgraded.Spec.LandingZoneRef.Name != "example-default" {
				t.Errorf("landing_zone_ref not carried over, got %+v", upgraded.Spec.LandingZoneRef)
			}
			if upgraded.Status.TenantName != "my-workspace.my-project.azure.germanywestcentral" {
				t.Errorf("status not carried over, got %+v", upgraded.Status)
			}
			if !upgraded.WaitForCompletion {
				t.Error("wait_for_completion not carried over")
			}
		})

		t.Run("keeps an existing requested_quotas map", func(t *testing.T) {
			var upgraded tenantResourceModel
			diags := UpgradeResourceStateFromJSON(t, &tenantResource{}, 1, tenantStateV1RequestedQuotas, &upgraded)
			if diags.HasError() {
				t.Fatalf("upgrade produced errors: %s", diags)
			}

			if len(upgraded.Spec.RequestedQuotas) != 1 || upgraded.Spec.RequestedQuotas["limits.cpu"].Value != 4 {
				t.Errorf("requested_quotas not preserved, got %v", upgraded.Spec.RequestedQuotas)
			}
			if upgraded.Spec.LandingZoneRef != nil {
				t.Errorf("null landing_zone_ref became %+v", upgraded.Spec.LandingZoneRef)
			}
			if upgraded.WaitForCompletion {
				t.Error("wait_for_completion flipped to true")
			}
		})

		t.Run("leaves requested_quotas null when neither field was set", func(t *testing.T) {
			var upgraded tenantResourceModel
			diags := UpgradeResourceStateFromJSON(t, &tenantResource{}, 1, tenantStateV1NoQuotas, &upgraded)
			if diags.HasError() {
				t.Fatalf("upgrade produced errors: %s", diags)
			}
			if upgraded.Spec.RequestedQuotas != nil {
				t.Errorf("requested_quotas became %v, want nil", upgraded.Spec.RequestedQuotas)
			}
		})
	})
}
