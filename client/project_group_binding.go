package client

import (
	"github.com/meshcloud/terraform-provider-meshstack/client/internal"
)

type MeshProjectGroupBinding struct {
	MeshProjectBinding
}

type MeshProjectGroupBindingClient struct {
	meshObject internal.MeshObjectClient[MeshProjectGroupBinding]
}

func newProjectGroupBindingClient(httpClient *internal.HttpClient) MeshProjectGroupBindingClient {
	return MeshProjectGroupBindingClient{
		meshObject: internal.NewMeshObjectClient[MeshProjectGroupBinding](httpClient, "v3", "meshprojectbindings", "groupbindings"),
	}
}

func (c MeshProjectGroupBindingClient) Read(name string) (*MeshProjectGroupBinding, error) {
	return c.meshObject.Get(name)
}

func (c MeshProjectGroupBindingClient) Create(binding *MeshProjectGroupBinding) (*MeshProjectGroupBinding, error) {
	return c.meshObject.Post(binding)
}

func (c MeshProjectGroupBindingClient) Delete(name string) error {
	return c.meshObject.Delete(name)
}
