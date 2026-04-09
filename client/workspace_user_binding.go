package client

import (
	"context"

	"github.com/meshcloud/terraform-provider-meshstack/client/internal"
)

type MeshWorkspaceUserBinding struct {
	MeshWorkspaceBinding
}

type MeshWorkspaceUserBindingClient interface {
	Read(ctx context.Context, name string) (*MeshWorkspaceUserBinding, error)
	Create(ctx context.Context, binding *MeshWorkspaceUserBinding) (*MeshWorkspaceUserBinding, error)
	Delete(ctx context.Context, name string) error
}

type meshWorkspaceUserBindingClient struct {
	meshObject internal.MeshObjectClient[MeshWorkspaceUserBinding]
}

func newWorkspaceUserBindingClient(ctx context.Context, httpClient *internal.HttpClient) MeshWorkspaceUserBindingClient {
	return meshWorkspaceUserBindingClient{internal.NewMeshObjectClient[MeshWorkspaceUserBinding](ctx, httpClient, "v2", "meshworkspacebindings", "userbindings")}
}

func (c meshWorkspaceUserBindingClient) Read(ctx context.Context, name string) (*MeshWorkspaceUserBinding, error) {
	return c.meshObject.Get(ctx, name)
}

func (c meshWorkspaceUserBindingClient) Create(ctx context.Context, binding *MeshWorkspaceUserBinding) (*MeshWorkspaceUserBinding, error) {
	return c.meshObject.Post(ctx, binding)
}

func (c meshWorkspaceUserBindingClient) Delete(ctx context.Context, name string) error {
	return c.meshObject.Delete(ctx, name)
}
