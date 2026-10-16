package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// v0.26.7 wrote building block definition state at schema version 0, with versions as a list.
const buildingBlockDefinitionStateWithVersionsList = `{
  "metadata": {"uuid": "bbd-uuid", "owned_by_workspace": "my-workspace"},
  "versions": [
    {"uuid": "uuid-1", "number": 1, "state": "RELEASED", "content_hash": "hash-1", "kind": "meshBuildingBlockDefinitionVersion", "workload_identity_federation": null},
    {"uuid": "uuid-2", "number": 2, "state": "DRAFT", "content_hash": "hash-2", "kind": "meshBuildingBlockDefinitionVersion", "workload_identity_federation": null}
  ]
}`

func TestBuildingBlockDefinitionUpgradeState(t *testing.T) {
	upgraded, diags := UpgradeResourceState(t, &buildingBlockDefinitionResource{}, 0, buildingBlockDefinitionStateWithVersionsList)
	require.False(t, diags.HasError(), "upgrade produced errors: %s", diags)

	var versions map[string]types.Object
	upgraded.Attribute("versions", &versions)

	uuids := map[string]string{}
	for key, version := range versions {
		uuid, ok := version.Attributes()["uuid"].(types.String)
		require.True(t, ok)
		uuids[key] = uuid.ValueString()
	}
	assert.Equal(t, map[string]string{"v1": "uuid-1", "v2": "uuid-2", "0": "uuid-1", "1": "uuid-2"}, uuids)

	var metadataUuid string
	upgraded.Attribute("metadata.uuid", &metadataUuid)
	assert.Equal(t, "bbd-uuid", metadataUuid)
}
