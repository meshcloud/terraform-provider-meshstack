package client

import (
	"context"

	"github.com/meshcloud/terraform-provider-meshstack/client/internal"
)

type MeshWorkspaceGroupBinding struct {
	MeshWorkspaceBinding
}

type MeshWorkspaceGroupBindingClient interface {
	Read(ctx context.Context, name string) (*MeshWorkspaceGroupBinding, error)
	Create(ctx context.Context, binding *MeshWorkspaceGroupBinding) (*MeshWorkspaceGroupBinding, error)
	Delete(ctx context.Context, name string) error
}

type meshWorkspaceGroupBindingClient struct {
	meshObject internal.MeshObjectClient[MeshWorkspaceGroupBinding]
}

func newWorkspaceGroupBindingClient(ctx context.Context, httpClient *internal.HttpClient) MeshWorkspaceGroupBindingClient {
	return meshWorkspaceGroupBindingClient{internal.NewMeshObjectClient[MeshWorkspaceGroupBinding](ctx, httpClient, "v2", "meshworkspacebindings", "groupbindings")}
}

func (c meshWorkspaceGroupBindingClient) Read(ctx context.Context, name string) (*MeshWorkspaceGroupBinding, error) {
	return c.meshObject.Get(ctx, name)
}

func (c meshWorkspaceGroupBindingClient) Create(ctx context.Context, binding *MeshWorkspaceGroupBinding) (*MeshWorkspaceGroupBinding, error) {
	return c.meshObject.Post(ctx, binding)
}

func (c meshWorkspaceGroupBindingClient) Delete(ctx context.Context, name string) error {
	return c.meshObject.Delete(ctx, name)
}
