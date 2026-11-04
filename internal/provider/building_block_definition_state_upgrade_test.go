package provider

import (
	_ "embed"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// As provider v0.26.7 wrote it, with versions as a list.
//
//go:embed testdata/bbd/state-v0.json
var buildingBlockDefinitionStateV0 string

func TestBuildingBlockDefinitionUpgradeState(t *testing.T) {
	t.Run("v0_to_v1", func(t *testing.T) {
		upgraded, diags := UpgradeResourceState(t, &buildingBlockDefinitionResource{}, 0, buildingBlockDefinitionStateV0)
		require.False(t, diags.HasError(), "upgrade produced errors: %s", diags)

		var versions map[string]types.Object
		upgraded.Attribute("versions", &versions)

		uuids := map[string]string{}
		for key, version := range versions {
			uuid, ok := version.Attributes()["uuid"].(types.String)
			require.True(t, ok)
			uuids[key] = uuid.ValueString()
		}
		assert.Equal(t, map[string]string{"v1": "uuid-1", "v2": "uuid-2"}, uuids)

		var metadataUuid string
		upgraded.Attribute("metadata.uuid", &metadataUuid)
		assert.Equal(t, "bbd-uuid", metadataUuid)
	})

	t.Run("v0_to_v1_rejects_versions_without_unique_number", func(t *testing.T) {
		for name, versions := range map[string]string{
			"no number":         `[{"uuid": "uuid-1", "number": null}]`,
			"fractional number": `[{"uuid": "uuid-1", "number": 1.5}]`,
			"duplicate number":  `[{"uuid": "uuid-1", "number": 1}, {"uuid": "uuid-2", "number": 1}]`,
		} {
			t.Run(name, func(t *testing.T) {
				_, diags := UpgradeResourceState(t, &buildingBlockDefinitionResource{}, 0, `{"versions": `+versions+`}`)
				require.True(t, diags.HasError())
				assert.Equal(t, "Cannot read a version number from the prior building block definition state", diags[0].Summary())
			})
		}
	})
}
