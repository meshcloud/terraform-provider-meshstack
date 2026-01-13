package client

import (
	"context"

	"github.com/meshcloud/terraform-provider-meshstack/client/internal"
)

type MeshProjectUserBinding struct {
	MeshProjectBinding
}

type MeshProjectUserBindingClient struct {
	meshObject internal.MeshObjectClient[MeshProjectUserBinding]
}

func newProjectUserBindingClient(ctx context.Context, httpClient *internal.HttpClient) MeshProjectUserBindingClient {
	return MeshProjectUserBindingClient{internal.NewMeshObjectClient[MeshProjectUserBinding](ctx, httpClient, "v3", "meshprojectbindings", "userbindings")}
}

func (c MeshProjectUserBindingClient) Read(ctx context.Context, name string) (*MeshProjectUserBinding, error) {
	return c.meshObject.Get(ctx, name)
}

func (c MeshProjectUserBindingClient) Create(ctx context.Context, binding *MeshProjectUserBinding) (*MeshProjectUserBinding, error) {
	return c.meshObject.Post(ctx, binding)
}

func (c MeshProjectUserBindingClient) Delete(ctx context.Context, name string) error {
	return c.meshObject.Delete(ctx, name)
}
