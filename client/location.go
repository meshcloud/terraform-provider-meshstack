package client

import (
	"context"

	"github.com/meshcloud/terraform-provider-meshstack/client/internal"
)

type MeshLocation struct {
	ApiVersion string               `json:"apiVersion" tfsdk:"api_version"`
	Metadata   MeshLocationMetadata `json:"metadata" tfsdk:"metadata"`
	Spec       MeshLocationSpec     `json:"spec" tfsdk:"spec"`
	Status     MeshLocationStatus   `json:"status" tfsdk:"status"`
}

type MeshLocationMetadata struct {
	Name string `json:"name" tfsdk:"name"`
	Uuid string `json:"uuid" tfsdk:"uuid"`
}

type MeshLocationSpec struct {
	DisplayName string `json:"displayName" tfsdk:"display_name"`
	Description string `json:"description" tfsdk:"description"`
}

type MeshLocationStatus struct {
	IsPublic bool `json:"isPublic" tfsdk:"is_public"`
}

type MeshLocationCreate struct {
	ApiVersion string                     `json:"apiVersion" tfsdk:"api_version"`
	Metadata   MeshLocationCreateMetadata `json:"metadata" tfsdk:"metadata"`
	Spec       MeshLocationSpec           `json:"spec" tfsdk:"spec"`
}

type MeshLocationCreateMetadata struct {
	Name string `json:"name" tfsdk:"name"`
}

type MeshLocationClient interface {
	Read(ctx context.Context, name string) (*MeshLocation, error)
	Create(ctx context.Context, location *MeshLocationCreate) (*MeshLocation, error)
	Update(ctx context.Context, name string, location *MeshLocationCreate) (*MeshLocation, error)
	Delete(ctx context.Context, name string) error
}

type meshLocationClient struct {
	meshObject internal.MeshObjectClient[MeshLocation]
}

func newLocationClient(ctx context.Context, httpClient *internal.HttpClient) MeshLocationClient {
	return meshLocationClient{internal.NewMeshObjectClient[MeshLocation](ctx, httpClient, "v1-preview")}
}

func (c meshLocationClient) Read(ctx context.Context, name string) (*MeshLocation, error) {
	return c.meshObject.Get(ctx, name)
}

func (c meshLocationClient) Create(ctx context.Context, location *MeshLocationCreate) (*MeshLocation, error) {
	return c.meshObject.Post(ctx, location)
}

func (c meshLocationClient) Update(ctx context.Context, name string, location *MeshLocationCreate) (*MeshLocation, error) {
	return c.meshObject.Put(ctx, name, location)
}

func (c meshLocationClient) Delete(ctx context.Context, name string) error {
	return c.meshObject.Delete(ctx, name)
}
