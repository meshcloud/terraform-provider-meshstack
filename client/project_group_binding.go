package client

import (
	"context"

	"github.com/meshcloud/terraform-provider-meshstack/client/internal"
)

type MeshProjectGroupBinding struct {
	MeshProjectBinding
}

type MeshProjectGroupBindingClient struct {
	meshObject internal.MeshObjectClient[MeshProjectGroupBinding]
}

func newProjectGroupBindingClient(ctx context.Context, httpClient *internal.HttpClient) MeshProjectGroupBindingClient {
	return MeshProjectGroupBindingClient{
		meshObject: internal.NewMeshObjectClient[MeshProjectGroupBinding](ctx, httpClient, "v3", "meshprojectbindings", "groupbindings"),
	}
}

func (c MeshProjectGroupBindingClient) Read(ctx context.Context, name string) (*MeshProjectGroupBinding, error) {
	return c.meshObject.Get(ctx, name)
}

func (c MeshProjectGroupBindingClient) Create(ctx context.Context, binding *MeshProjectGroupBinding) (*MeshProjectGroupBinding, error) {
	return c.meshObject.Post(ctx, binding)
}

func (c MeshProjectGroupBindingClient) Delete(ctx context.Context, name string) error {
	return c.meshObject.Delete(ctx, name)
}
