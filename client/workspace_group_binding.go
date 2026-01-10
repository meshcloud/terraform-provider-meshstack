package client

import (
	"github.com/meshcloud/terraform-provider-meshstack/client/internal"
)

type MeshWorkspaceGroupBinding struct {
	MeshWorkspaceBinding
}

type MeshWorkspaceGroupBindingClient struct {
	meshObject internal.MeshObjectClient[MeshWorkspaceGroupBinding]
}

func newWorkspaceGroupBindingClient(httpClient *internal.HttpClient) MeshWorkspaceGroupBindingClient {
	return MeshWorkspaceGroupBindingClient{
		meshObject: internal.NewMeshObjectClient[MeshWorkspaceGroupBinding](httpClient, "v2", "meshworkspacebindings", "groupbindings"),
	}
}

func (c MeshWorkspaceGroupBindingClient) Read(name string) (*MeshWorkspaceGroupBinding, error) {
	return c.meshObject.Get(name)
}

func (c MeshWorkspaceGroupBindingClient) Create(binding *MeshWorkspaceGroupBinding) (*MeshWorkspaceGroupBinding, error) {
	return c.meshObject.Post(binding)
}

func (c MeshWorkspaceGroupBindingClient) Delete(name string) error {
	return c.meshObject.Delete(name)
}
