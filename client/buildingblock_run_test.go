package client

import (
	"context"
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/meshcloud/terraform-provider-meshstack/client/internal"
)

func TestBuildingBlockRunClientUsesV1Transport(t *testing.T) {
	rootURL, err := url.Parse("https://meshstack.example")
	require.NoError(t, err)

	typedClient := newBuildingBlockRunClient(context.Background(), &internal.HttpClient{
		RootUrl: rootURL,
	})

	concrete, ok := typedClient.(meshBuildingBlockRunClient)
	require.True(t, ok)
	assert.Equal(t, "v1", concrete.meshObject.ApiVersion)
	assert.Equal(t, MeshObjectKind.BuildingBlockRun, concrete.meshObject.Kind)
	assert.Equal(t, "https://meshstack.example/api/meshobjects/meshbuildingblockruns", concrete.meshObject.ApiUrl.String())
}
