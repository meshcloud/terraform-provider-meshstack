package provider

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/meshcloud/terraform-provider-meshstack/client"
)

// Captured from meshstack_building_block state as provider v0.24.3 wrote it at schema version 0.
const buildingBlockStateLegacy = `{
  "metadata": {
    "uuid": "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa",
    "owned_by_workspace": "my-workspace"
  },
  "spec": {
    "display_name": "my-workspace-building-block",
    "building_block_definition_version_ref": {
      "uuid": "bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb",
      "kind": "meshBuildingBlockDefinitionVersion",
      "content_hash": null
    },
    "target_ref": {"kind": "meshWorkspace", "uuid": null, "name": "my-workspace"},
    "inputs": {},
    "parent_building_blocks": [
      {
        "buildingblock_uuid": "cccccccc-cccc-cccc-cccc-cccccccccccc",
        "definition_uuid": "dddddddd-dddd-dddd-dddd-dddddddddddd"
      }
    ]
  },
  "status": {
    "status": "SUCCEEDED",
    "force_purge": false,
    "outputs": {},
    "latest_run_uuid": "eeeeeeee-eeee-eeee-eeee-eeeeeeeeeeee",
    "latest_dry_run_uuid": null
  },
  "all_inputs": {},
  "wait_for_completion": true,
  "purge_on_delete": false,
  "timeouts": null
}`

func TestBuildingBlockUpgradeState(t *testing.T) {
	if _, ok := buildingBlockSchemaV0Once().Attributes["ref"]; ok {
		t.Error("prior v0 schema declares ref; state written before v0.24.4 has no such key")
	}
	if _, ok := ResourceSchemaForTest(t, &buildingBlockResource{}).Attributes["ref"]; !ok {
		t.Fatal("current schema lost its ref attribute")
	}

	upgraded, diags := UpgradeResourceState(t, &buildingBlockResource{}, 0, buildingBlockStateLegacy)
	require.Falsef(t, diags.HasError(), "upgrade produced errors: %s", diags)

	var refs []client.UuidRef
	upgraded.Attribute("spec.parent_building_block_refs", &refs)
	assert.Equal(t, []client.UuidRef{{
		Kind: client.MeshObjectKind.BuildingBlock,
		Uuid: "cccccccc-cccc-cccc-cccc-cccccccccccc",
	}}, refs, "the flat parent pair becomes a ref under the new attribute name, and the parent's definition uuid is dropped")

	var ref client.UuidRef
	upgraded.Attribute("ref", &ref)
	assert.Equal(t, client.UuidRef{
		Kind: client.MeshObjectKind.BuildingBlock,
		Uuid: "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa",
	}, ref, "ref is recomputed from metadata.uuid")

	// The upgrader copies every other attribute, so a sample of them must come through unchanged.
	var displayName, versionUuid string
	upgraded.Attribute("spec.display_name", &displayName)
	upgraded.Attribute("spec.building_block_definition_version_ref.uuid", &versionUuid)
	assert.Equal(t, "my-workspace-building-block", displayName)
	assert.Equal(t, "bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb", versionUuid)
}
