package client

import (
	"github.com/meshcloud/terraform-provider-meshstack/client/internal"
)

type MeshProjectUserBinding struct {
	MeshProjectBinding
}

type MeshProjectUserBindingClient struct {
	meshObject internal.MeshObjectClient[MeshProjectUserBinding]
}

func newProjectUserBindingClient(httpClient *internal.HttpClient) MeshProjectUserBindingClient {
	return MeshProjectUserBindingClient{
		meshObject: internal.NewMeshObjectClient[MeshProjectUserBinding](httpClient, "v3", "meshprojectbindings", "userbindings"),
	}
}

func (c MeshProjectUserBindingClient) Read(name string) (*MeshProjectUserBinding, error) {
	return c.meshObject.Get(name)
}

func (c MeshProjectUserBindingClient) Create(binding *MeshProjectUserBinding) (*MeshProjectUserBinding, error) {
	return c.meshObject.Post(binding)
}

func (c MeshProjectUserBindingClient) Delete(name string) error {
	return c.meshObject.Delete(name)
}
