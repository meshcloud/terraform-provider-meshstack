package clientmock

import (
	"context"

	"github.com/meshcloud/terraform-provider-meshstack/client"
)

type MeshInfoClient struct{}

func (m MeshInfoClient) Read(_ context.Context) (*client.MeshInfo, error) {
	return &client.MeshInfo{
		// The mock client factory (see ApplyAndTest) never sees the provider's actual configured
		// endpoint, since it bypasses newProviderClient entirely.
		Endpoint:          "http://localhost:8080",
		Version:           "2026.30.0",
		IsFourEyesEnabled: false,
		Metadata: map[string]string{
			"test": "test",
		},
		AdminWorkspaceIdentifier: "demo-partner",
	}, nil
}
