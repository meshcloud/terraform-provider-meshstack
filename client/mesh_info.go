package client

import (
	"context"
	"fmt"

	"github.com/meshcloud/terraform-provider-meshstack/client/internal"
	"github.com/meshcloud/terraform-provider-meshstack/client/version"
)

// MeshInfo describes the meshStack instance the provider is configured against: the endpoint from
// the provider configuration, plus metadata from the public, unauthenticated /mesh/info endpoint.
type MeshInfo struct {
	Endpoint                 string            `tfsdk:"endpoint"`
	Version                  string            `tfsdk:"version"`
	IsFourEyesEnabled        bool              `tfsdk:"is_four_eyes_enabled"`
	Metadata                 map[string]string `tfsdk:"metadata"`
	AdminWorkspaceIdentifier string            `tfsdk:"admin_workspace_identifier"`
}

// meshInfoDto is the raw /mesh/info response shape.
type meshInfoDto struct {
	Version                  version.Version   `json:"version"`
	Is4EPEnabled             bool              `json:"is4EPEnabled"`
	Metadata                 map[string]string `json:"metadata"`
	AdminWorkspaceIdentifier string            `json:"adminWorkspaceIdentifier"`
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
	dto, err := fetchMeshInfo(ctx, c.httpClient)
	if err != nil {
		return nil, err
	}

	return &MeshInfo{
		Endpoint:                 c.httpClient.RootUrl.String(),
		Version:                  dto.Version.String(),
		IsFourEyesEnabled:        dto.Is4EPEnabled,
		Metadata:                 dto.Metadata,
		AdminWorkspaceIdentifier: dto.AdminWorkspaceIdentifier,
	}, nil
}

func fetchMeshInfo(ctx context.Context, httpClient internal.HttpClient) (meshInfoDto, error) {
	meshInfoEndpoint := httpClient.RootUrl.JoinPath("/mesh/info")
	dto, err := internal.DoRequest[meshInfoDto](ctx, httpClient, "GET", meshInfoEndpoint)
	if err != nil {
		return meshInfoDto{}, fmt.Errorf("failed to retrieve meshStack instance information from %s endpoint: %w", meshInfoEndpoint, err)
	}
	return dto, nil
}
