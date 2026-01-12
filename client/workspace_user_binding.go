package client

import (
	"context"

	"github.com/meshcloud/terraform-provider-meshstack/client/internal"
)

type MeshWorkspaceUserBinding struct {
	MeshWorkspaceBinding
}

type MeshWorkspaceUserBindingClient struct {
	meshObject internal.MeshObjectClient[MeshWorkspaceUserBinding]
}

func newWorkspaceUserBindingClient(ctx context.Context, httpClient *internal.HttpClient) MeshWorkspaceUserBindingClient {
	return MeshWorkspaceUserBindingClient{
		meshObject: internal.NewMeshObjectClient[MeshWorkspaceUserBinding](ctx, httpClient, "v2", "meshworkspacebindings", "userbindings"),
	}
}

func (c MeshWorkspaceUserBindingClient) Read(ctx context.Context, name string) (*MeshWorkspaceUserBinding, error) {
	return c.meshObject.Get(ctx, name)
}

func (c MeshWorkspaceUserBindingClient) Create(ctx context.Context, binding *MeshWorkspaceUserBinding) (*MeshWorkspaceUserBinding, error) {
	return c.meshObject.Post(ctx, binding)
}

func (c MeshWorkspaceUserBindingClient) Delete(ctx context.Context, name string) error {
	return c.meshObject.Delete(ctx, name)
}
