package client

import (
	"context"

	"github.com/meshcloud/terraform-provider-meshstack/client/internal"
)

type MeshApiKey struct {
	Metadata MeshApiKeyMetadata `json:"metadata"`
	Spec     MeshApiKeySpec     `json:"spec"`
	Status   *MeshApiKeyStatus  `json:"status,omitempty"`
}

type MeshApiKeyMetadata struct {
	Uuid             *string `json:"uuid,omitempty"`
	OwnedByWorkspace string  `json:"ownedByWorkspace"`
}

type MeshApiKeySpec struct {
	DisplayName string   `json:"displayName"`
	Authorities []string `json:"authorities"`
	ExpiresAt   *string  `json:"expiresAt,omitempty"`
}

type MeshApiKeyStatus struct {
	Token *string `json:"token,omitempty"`
}

type MeshApiKeyCreate struct {
	Metadata MeshApiKeyCreateMetadata `json:"metadata"`
	Spec     MeshApiKeySpec           `json:"spec"`
}

type MeshApiKeyCreateMetadata struct {
	Uuid             *string `json:"uuid,omitempty"`
	OwnedByWorkspace string  `json:"ownedByWorkspace"`
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
	return meshApiKeyClient{internal.NewMeshObjectClient[MeshApiKey](ctx, httpClient, "v1-preview")}
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
