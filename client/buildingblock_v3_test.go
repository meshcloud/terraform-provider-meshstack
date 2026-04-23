package client

import (
	"context"
	"encoding/json"
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/meshcloud/terraform-provider-meshstack/client/internal"
)

func TestBuildingBlockV3ClientUsesV2PreviewTransport(t *testing.T) {
	rootURL, err := url.Parse("https://meshstack.example")
	require.NoError(t, err)

	typedClient := newBuildingBlockV3Client(context.Background(), &internal.HttpClient{
		RootUrl: rootURL,
	})

	concrete, ok := typedClient.(meshBuildingBlockV3Client)
	require.True(t, ok)
	assert.Equal(t, "v2-preview", concrete.meshObject.ApiVersion)
	assert.Equal(t, MeshObjectKind.BuildingBlock, concrete.meshObject.Kind)
	assert.Equal(t, "https://meshstack.example/api/meshobjects/meshbuildingblocks", concrete.meshObject.ApiUrl.String())
}

func TestBuildingBlockV3SpecMarshalJSONUsesInputList(t *testing.T) {
	spec := MeshBuildingBlockV3Spec{
		DisplayName: "bb",
		Inputs: map[string]MeshBuildingBlockV3InputValue{
			"name": {
				Value:     "my-name",
				ValueType: MESH_BUILDING_BLOCK_IO_TYPE_STRING,
			},
			"size": {
				Value:     16,
				ValueType: MESH_BUILDING_BLOCK_IO_TYPE_INTEGER,
			},
		},
	}

	payload, err := json.Marshal(spec)
	require.NoError(t, err)

	var decoded struct {
		Inputs []MeshBuildingBlockIO `json:"inputs"`
	}
	require.NoError(t, json.Unmarshal(payload, &decoded))
	require.Len(t, decoded.Inputs, 2)

	found := map[string]MeshBuildingBlockIO{}
	for _, input := range decoded.Inputs {
		found[input.Key] = input
	}
	require.Equal(t, "my-name", found["name"].Value)
	require.Equal(t, MESH_BUILDING_BLOCK_IO_TYPE_STRING, found["name"].ValueType)
	require.InEpsilon(t, float64(16), found["size"].Value, 0.00001)
	require.Equal(t, MESH_BUILDING_BLOCK_IO_TYPE_INTEGER, found["size"].ValueType)
}

func TestBuildingBlockV3SpecUnmarshalJSONReadsInputList(t *testing.T) {
	payload := []byte(`{
		"displayName":"bb",
		"inputs":[
			{"key":"enabled","value":true,"valueType":"BOOLEAN"},
			{"key":"name","value":"my-name","valueType":"STRING"}
		]
	}`)

	var spec MeshBuildingBlockV3Spec
	err := json.Unmarshal(payload, &spec)
	require.NoError(t, err)
	require.Len(t, spec.Inputs, 2)
	require.Equal(t, true, spec.Inputs["enabled"].Value)
	require.Equal(t, MESH_BUILDING_BLOCK_IO_TYPE_BOOLEAN, spec.Inputs["enabled"].ValueType)
	require.Equal(t, "my-name", spec.Inputs["name"].Value)
	require.Equal(t, MESH_BUILDING_BLOCK_IO_TYPE_STRING, spec.Inputs["name"].ValueType)
}
