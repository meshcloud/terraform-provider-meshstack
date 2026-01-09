package client

type MeshProjectUserBinding struct {
	MeshProjectBinding
}

type MeshProjectUserBindingClient struct {
	meshObjectClient[MeshProjectUserBinding]
}

func newProjectUserBindingClient(c *httpClient) MeshProjectUserBindingClient {
	return MeshProjectUserBindingClient{newMeshObjectClient[MeshProjectUserBinding](c, "v3", "meshprojectbindings", "userbindings")}
}

func (c MeshProjectUserBindingClient) Read(name string) (*MeshProjectUserBinding, error) {
	return c.get(name)
}

func (c MeshProjectUserBindingClient) Create(binding *MeshProjectUserBinding) (*MeshProjectUserBinding, error) {
	return c.post(binding)
}

func (c MeshProjectUserBindingClient) Delete(name string) error {
	return c.delete(name)
}
