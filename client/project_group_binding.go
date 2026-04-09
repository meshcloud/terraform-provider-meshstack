package client

import (
	"context"

	"github.com/meshcloud/terraform-provider-meshstack/client/internal"
)

type MeshProjectGroupBinding struct {
	MeshProjectBinding
}

type MeshProjectGroupBindingClient interface {
	Read(ctx context.Context, name string) (*MeshProjectGroupBinding, error)
	Create(ctx context.Context, binding *MeshProjectGroupBinding) (*MeshProjectGroupBinding, error)
	Delete(ctx context.Context, name string) error
}

type meshProjectGroupBindingClient struct {
	meshObject internal.MeshObjectClient[MeshProjectGroupBinding]
}

func newProjectGroupBindingClient(ctx context.Context, httpClient *internal.HttpClient) MeshProjectGroupBindingClient {
	return meshProjectGroupBindingClient{internal.NewMeshObjectClient[MeshProjectGroupBinding](ctx, httpClient, "v3", "meshprojectbindings", "groupbindings")}
}

func (c meshProjectGroupBindingClient) Read(ctx context.Context, name string) (*MeshProjectGroupBinding, error) {
	return c.meshObject.Get(ctx, name)
}

func (c meshProjectGroupBindingClient) Create(ctx context.Context, binding *MeshProjectGroupBinding) (*MeshProjectGroupBinding, error) {
	return c.meshObject.Post(ctx, binding)
}

func (c meshProjectGroupBindingClient) Delete(ctx context.Context, name string) error {
	return c.meshObject.Delete(ctx, name)
}
