package clientmock

import (
	"context"

	"github.com/meshcloud/meshstack-cli/client"
)

type MeshInfoClient struct{}

func (m MeshInfoClient) Read(_ context.Context) (client.MeshInfo, error) {
	return client.MeshInfo{
		Version:      "2026.30.0",
		Is4EPEnabled: true,
		Metadata: map[string]string{
			"test": "test",
		},
		AdminWorkspaceIdentifier: "demo-partner",
	}, nil
}
