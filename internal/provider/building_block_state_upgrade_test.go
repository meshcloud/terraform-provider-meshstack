package provider

import (
	_ "embed"
	"testing"

	"github.com/meshcloud/meshstack-cli/client"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// As provider v0.24.3 wrote it.
//
//go:embed testdata/bb/state-v0.json
var buildingBlockStateV0 string

func TestBuildingBlockUpgradeState(t *testing.T) {
	t.Run("v0_to_v1", func(t *testing.T) {
		if _, ok := buildingBlockSchemaV0Once().Attributes["ref"]; ok {
			t.Error("prior v0 schema declares ref; state written before v0.24.4 has no such key")
		}
		if _, ok := ResourceSchemaForTest(t, &buildingBlockResource{}).Attributes["ref"]; !ok {
			t.Fatal("current schema lost its ref attribute")
		}

		upgraded, diags := UpgradeResourceState(t, &buildingBlockResource{}, 0, buildingBlockStateV0)
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
	})
}
