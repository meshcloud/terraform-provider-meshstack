package client

import (
	"context"

	"github.com/meshcloud/terraform-provider-meshstack/client/internal"
)

type MeshApiKey struct {
	Metadata MeshApiKeyMetadata `json:"metadata" tfsdk:"metadata"`
	Spec     MeshApiKeySpec     `json:"spec" tfsdk:"spec"`
	Token    *string            `json:"token,omitempty" tfsdk:"token"`
}

type MeshApiKeyMetadata struct {
	Uuid             *string `json:"uuid,omitempty" tfsdk:"uuid"`
	Name             string  `json:"name" tfsdk:"name"`
	OwnedByWorkspace string  `json:"ownedByWorkspace" tfsdk:"owned_by_workspace"`
	CreatedOn        string  `json:"createdOn" tfsdk:"created_on"`
}

type MeshApiKeySpec struct {
	Authorities []string `json:"authorities" tfsdk:"authorities"`
	ExpiryDate  string   `json:"expiryDate" tfsdk:"expiry_date"`
}

type MeshApiKeyCreate struct {
	Metadata MeshApiKeyCreateMetadata `json:"metadata" tfsdk:"metadata"`
	Spec     MeshApiKeySpec           `json:"spec" tfsdk:"spec"`
}

type MeshApiKeyCreateMetadata struct {
	Name             string `json:"name" tfsdk:"name"`
	OwnedByWorkspace string `json:"ownedByWorkspace" tfsdk:"owned_by_workspace"`
}

type MeshApiKeyClient interface {
	Create(ctx context.Context, apiKey *MeshApiKeyCreate) (*MeshApiKey, error)
	Read(ctx context.Context, uuid string) (*MeshApiKey, error)
	Update(ctx context.Context, uuid string, apiKey *MeshApiKeyCreate) (*MeshApiKey, error)
	Delete(ctx context.Context, uuid string) error
}

type meshApiKeyClient struct {
	meshObject internal.MeshObjectClient[MeshApiKey]
}

func newApiKeyClient(ctx context.Context, httpClient *internal.HttpClient) MeshApiKeyClient {
	return meshApiKeyClient{internal.NewMeshObjectClient[MeshApiKey](ctx, httpClient, "v1")}
}

func (c meshApiKeyClient) Create(ctx context.Context, apiKey *MeshApiKeyCreate) (*MeshApiKey, error) {
	return c.meshObject.Post(ctx, apiKey)
}

func (c meshApiKeyClient) Read(ctx context.Context, uuid string) (*MeshApiKey, error) {
	return c.meshObject.Get(ctx, uuid)
}

func (c meshApiKeyClient) Update(ctx context.Context, uuid string, apiKey *MeshApiKeyCreate) (*MeshApiKey, error) {
	return c.meshObject.Put(ctx, uuid, apiKey)
}

func (c meshApiKeyClient) Delete(ctx context.Context, uuid string) error {
	return c.meshObject.Delete(ctx, uuid)
}
