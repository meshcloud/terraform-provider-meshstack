package client

import (
	"context"

	"github.com/meshcloud/terraform-provider-meshstack/client/internal"
	"github.com/meshcloud/terraform-provider-meshstack/client/types"
)

type MeshApiKey struct {
	Metadata MeshApiKeyMetadata `json:"metadata" tfsdk:"metadata"`
	Spec     MeshApiKeySpec     `json:"spec" tfsdk:"spec"`
	Status   *MeshApiKeyStatus  `json:"status,omitempty" tfsdk:"status"`
}

type MeshApiKeyMetadata struct {
	Uuid             *string `json:"uuid,omitempty" tfsdk:"uuid"`
	OwnedByWorkspace string  `json:"ownedByWorkspace" tfsdk:"owned_by_workspace"`
}

type MeshApiKeySpec struct {
	DisplayName string                   `json:"displayName" tfsdk:"display_name"`
	Permissions types.Set[ApiPermission] `json:"permissions" tfsdk:"permissions"`
	ExpiresAt   *string                  `json:"expiresAt,omitempty" tfsdk:"expires_at"`
}

type MeshApiKeyStatus struct {
	ClientId     string  `json:"clientId" tfsdk:"client_id"`
	ClientSecret *string `json:"clientSecret,omitempty" tfsdk:"client_secret"`
}

type MeshApiKeyClient interface {
	Create(ctx context.Context, apiKey *MeshApiKey) (*MeshApiKey, error)
	Read(ctx context.Context, uuid string) (*MeshApiKey, error)
	Update(ctx context.Context, uuid string, apiKey *MeshApiKey) (*MeshApiKey, error)
	Delete(ctx context.Context, uuid string) error
}

type meshApiKeyClient struct {
	meshObject internal.MeshObjectClient[MeshApiKey]
}

func newApiKeyClient(ctx context.Context, httpClient internal.HttpClient) MeshApiKeyClient {
	return meshApiKeyClient{internal.NewMeshObjectClient[MeshApiKey](ctx, httpClient, "v1-preview")}
}

func (c meshApiKeyClient) Create(ctx context.Context, apiKey *MeshApiKey) (*MeshApiKey, error) {
	return c.meshObject.Post(ctx, apiKey)
}

func (c meshApiKeyClient) Read(ctx context.Context, uuid string) (*MeshApiKey, error) {
	return c.meshObject.Get(ctx, uuid)
}

func (c meshApiKeyClient) Update(ctx context.Context, uuid string, apiKey *MeshApiKey) (*MeshApiKey, error) {
	return c.meshObject.Put(ctx, uuid, apiKey)
}

func (c meshApiKeyClient) Delete(ctx context.Context, uuid string) error {
	return c.meshObject.Delete(ctx, uuid)
}
