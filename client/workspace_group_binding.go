package client

import (
	"context"

	"github.com/meshcloud/terraform-provider-meshstack/client/internal"
)

type MeshWorkspaceGroupBinding struct {
	MeshWorkspaceBinding
}

type MeshWorkspaceGroupBindingClient struct {
	meshObject internal.MeshObjectClient[MeshWorkspaceGroupBinding]
}

func newWorkspaceGroupBindingClient(ctx context.Context, httpClient *internal.HttpClient) MeshWorkspaceGroupBindingClient {
	return MeshWorkspaceGroupBindingClient{
		meshObject: internal.NewMeshObjectClient[MeshWorkspaceGroupBinding](ctx, httpClient, "v2", "meshworkspacebindings", "groupbindings"),
	}
}

func (c MeshWorkspaceGroupBindingClient) Read(ctx context.Context, name string) (*MeshWorkspaceGroupBinding, error) {
	return c.meshObject.Get(ctx, name)
}

func (c MeshWorkspaceGroupBindingClient) Create(ctx context.Context, binding *MeshWorkspaceGroupBinding) (*MeshWorkspaceGroupBinding, error) {
	return c.meshObject.Post(ctx, binding)
}

func (c MeshWorkspaceGroupBindingClient) Delete(ctx context.Context, name string) error {
	return c.meshObject.Delete(ctx, name)
}
