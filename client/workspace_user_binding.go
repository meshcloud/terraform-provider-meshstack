package client

import (
	"github.com/meshcloud/terraform-provider-meshstack/client/internal"
)

type MeshWorkspaceUserBinding struct {
	MeshWorkspaceBinding
}

type MeshWorkspaceUserBindingClient struct {
	meshObject internal.MeshObjectClient[MeshWorkspaceUserBinding]
}

func newWorkspaceUserBindingClient(httpClient *internal.HttpClient) MeshWorkspaceUserBindingClient {
	return MeshWorkspaceUserBindingClient{
		meshObject: internal.NewMeshObjectClient[MeshWorkspaceUserBinding](httpClient, "v2", "meshworkspacebindings", "userbindings"),
	}
}

func (c MeshWorkspaceUserBindingClient) Read(name string) (*MeshWorkspaceUserBinding, error) {
	return c.meshObject.Get(name)
}

func (c MeshWorkspaceUserBindingClient) Create(binding *MeshWorkspaceUserBinding) (*MeshWorkspaceUserBinding, error) {
	return c.meshObject.Post(binding)
}

func (c MeshWorkspaceUserBindingClient) Delete(name string) error {
	return c.meshObject.Delete(name)
}
