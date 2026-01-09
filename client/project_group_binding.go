package client

type MeshProjectGroupBinding = MeshProjectBinding

type MeshProjectGroupBindingClient struct {
	meshObjectClient[MeshProjectBinding]
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
