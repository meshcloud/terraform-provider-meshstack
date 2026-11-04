package provider

import (
	_ "embed"
	"testing"
)

// As provider v0.23.3 wrote it, without a `ref` key and with tags as a set.
//
//go:embed testdata/landingzone/state-v0.json
var landingZoneStateV0 string

func TestLandingZoneUpgradeState(t *testing.T) {
	t.Run("v0_to_v1", func(t *testing.T) {
		if _, ok := landingZoneSchemaV0Once().Attributes["ref"]; ok {
			t.Error("prior v0 schema declares ref; state written before v0.24.0 has no such key")
		}
		if _, ok := ResourceSchemaForTest(t, &landingZoneResource{}).Attributes["ref"]; !ok {
			t.Fatal("current schema lost its ref attribute")
		}

		var upgraded landingZoneModel
		diags := UpgradeResourceStateFromJSON(t, &landingZoneResource{}, 0, landingZoneStateV0, &upgraded)
		if diags.HasError() {
			t.Fatalf("upgrade produced errors: %s", diags)
		}

		if upgraded.Ref.Name != "example-default" || upgraded.Ref.Kind != "meshLandingZone" {
			t.Errorf("ref not recomputed from metadata.name, got %+v", upgraded.Ref)
		}
		if got := upgraded.Metadata.Tags["env"]; len(got) != 1 || got[0] != "prod" {
			t.Errorf("tags not carried over as a list, got %v", upgraded.Metadata.Tags)
		}
		if upgraded.Spec.DisplayName != "Example Default" {
			t.Errorf("spec not carried over, got %+v", upgraded.Spec)
		}
		// spec.restricted postdates this state, so it decodes as null and must land on the schema
		// default.
		if upgraded.Spec.Restricted {
			t.Error("spec.restricted absent from legacy state should upgrade to false")
		}
	})
}
