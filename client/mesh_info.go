package client

import (
	"context"
	"fmt"

	"github.com/meshcloud/terraform-provider-meshstack/client/internal"
)

// MeshInfo describes the meshStack instance the provider is configured against: the endpoint from
// the provider configuration, plus metadata from the public, unauthenticated /mesh/info endpoint.
type MeshInfo struct {
	Endpoint                 string            `tfsdk:"endpoint" json:"-"`
	Version                  string            `tfsdk:"version" json:"version"`
	IsFourEyesEnabled        bool              `tfsdk:"is_four_eyes_enabled" json:"is4EPEnabled"`
	Metadata                 map[string]string `tfsdk:"metadata" json:"metadata"`
	AdminWorkspaceIdentifier string            `tfsdk:"admin_workspace_identifier" json:"adminWorkspaceIdentifier"`
}

type MeshInfoClient interface {
	Read(ctx context.Context) (*MeshInfo, error)
}

type meshInfoClient struct {
	httpClient internal.HttpClient
}

func newMeshInfoClient(httpClient internal.HttpClient) MeshInfoClient {
	return meshInfoClient{httpClient: httpClient}
}

func (c meshInfoClient) Read(ctx context.Context) (*MeshInfo, error) {
	meshInfoEndpoint := c.httpClient.RootUrl.JoinPath("/mesh/info")
	info, err := internal.DoRequest[MeshInfo](ctx, c.httpClient, "GET", meshInfoEndpoint)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve meshStack instance information from %s endpoint: %w", meshInfoEndpoint, err)
	}

	info.Endpoint = c.httpClient.RootUrl.String()
	return &info, nil
}
