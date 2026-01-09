package client

type MeshProjectGroupBinding struct {
	MeshProjectBinding
}

type MeshProjectGroupBindingClient struct {
	meshObjectClient[MeshProjectGroupBinding]
}

func newProjectGroupBindingClient(c *httpClient) MeshProjectGroupBindingClient {
	return MeshProjectGroupBindingClient{newMeshObjectClient[MeshProjectGroupBinding](c, "v3", "meshprojectbindings", "groupbindings")}
}

func (c MeshProjectGroupBindingClient) Read(name string) (*MeshProjectGroupBinding, error) {
	return c.get(name)
}

func (c MeshProjectGroupBindingClient) Create(binding *MeshProjectGroupBinding) (*MeshProjectGroupBinding, error) {
	return c.post(binding)
}

func (c MeshProjectGroupBindingClient) Delete(name string) error {
	return c.delete(name)
}
