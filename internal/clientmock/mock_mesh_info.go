package clientmock

import (
	"context"

	"github.com/meshcloud/terraform-provider-meshstack/client"
)

// MockEndpoint is the fixed endpoint the mock client reports for meshstack_instance: the mock
// client factory (see ApplyAndTest) never sees the provider's actual configured endpoint, since it
// bypasses newProviderClient entirely.
const MockEndpoint = "http://localhost:8080"

const MockMeshVersion = "2026.30.0"

const mockAdminWorkspaceIdentifier = "demo-partner"

type MeshInfoClient struct{}

func (m MeshInfoClient) Read(_ context.Context) (*client.MeshInfo, error) {
	return &client.MeshInfo{
		Endpoint:          MockEndpoint,
		Version:           MockMeshVersion,
		IsFourEyesEnabled: false,
		Metadata: map[string]string{
			"test": "test",
		},
		AdminWorkspaceIdentifier: mockAdminWorkspaceIdentifier,
	}, nil
}
