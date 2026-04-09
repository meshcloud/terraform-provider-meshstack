package client

import (
	"context"

	"github.com/meshcloud/terraform-provider-meshstack/client/internal"
)

type MeshProjectUserBinding struct {
	MeshProjectBinding
}

type MeshProjectUserBindingClient interface {
	Read(ctx context.Context, name string) (*MeshProjectUserBinding, error)
	Create(ctx context.Context, binding *MeshProjectUserBinding) (*MeshProjectUserBinding, error)
	Delete(ctx context.Context, name string) error
}

type meshProjectUserBindingClient struct {
	meshObject internal.MeshObjectClient[MeshProjectUserBinding]
}

func newProjectUserBindingClient(ctx context.Context, httpClient *internal.HttpClient) MeshProjectUserBindingClient {
	return meshProjectUserBindingClient{internal.NewMeshObjectClient[MeshProjectUserBinding](ctx, httpClient, "v3", "meshprojectbindings", "userbindings")}
}

func (c meshProjectUserBindingClient) Read(ctx context.Context, name string) (*MeshProjectUserBinding, error) {
	return c.meshObject.Get(ctx, name)
}

func (c meshProjectUserBindingClient) Create(ctx context.Context, binding *MeshProjectUserBinding) (*MeshProjectUserBinding, error) {
	return c.meshObject.Post(ctx, binding)
}

func (c meshProjectUserBindingClient) Delete(ctx context.Context, name string) error {
	return c.meshObject.Delete(ctx, name)
}
