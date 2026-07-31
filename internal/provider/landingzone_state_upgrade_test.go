package provider

import (
	"testing"
)

// v0.23.3 wrote landing zone state at schema version 0 without a `ref` key (added in v0.24.0) and
// with tags as a set. Reading it back must not choke on the missing ref.
const landingZoneStateV0WithoutRef = `{
  "metadata": {
    "name": "example-default",
    "owned_by_workspace": "my-workspace",
    "tags": {"env": ["prod"]}
  },
  "spec": {
    "display_name": "Example Default",
    "description": "Default landing zone for example projects.",
    "automate_deletion_approval": true,
    "automate_deletion_replication": true,
    "info_link": null,
    "platform_ref": {"uuid": "a2b7cf9c-8e2a-4b0d-9e6f-1c4b9de3a111", "kind": "meshPlatform"},
    "platform_properties": {
      "type": "custom",
      "aws": null, "aks": null, "azure": null, "azurerg": null,
      "custom": {}, "gcp": null, "kubernetes": null, "openshift": null
    },
    "quotas": [],
    "mandatory_building_block_refs": [
      {"uuid": "6d1f0b3e-5a44-4d1c-90a2-7b8e2f0c5d22", "kind": "meshBuildingBlockDefinition"}
    ],
    "recommended_building_block_refs": []
  },
  "status": {"disabled": false, "restricted": false}
}`

func TestLandingZoneUpgradeStateV0(t *testing.T) {
	// The prior schema must not carry `ref`, but the live schema still must.
	if _, ok := landingZoneSchemaV0Once().Attributes["ref"]; ok {
		t.Error("prior v0 schema declares ref; state written before v0.24.0 has no such key")
	}
	if _, ok := ResourceSchemaForTest(t, &landingZoneResource{}).Attributes["ref"]; !ok {
		t.Fatal("current schema lost its ref attribute")
	}

	var upgraded landingZoneModel
	diags := UpgradeResourceStateFromJSON(t, &landingZoneResource{}, 0, landingZoneStateV0WithoutRef, &upgraded)
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
}
