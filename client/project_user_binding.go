package client

type MeshProjectUserBinding = MeshProjectBinding

type MeshProjectUserBindingClient struct {
	meshObjectClient[MeshProjectBinding]
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
