package client

type MeshWorkspaceGroupBinding = MeshWorkspaceBinding

type MeshWorkspaceGroupBindingClient struct {
	meshObjectClient[MeshWorkspaceBinding]
}

func (c MeshWorkspaceGroupBindingClient) Read(name string) (*MeshWorkspaceGroupBinding, error) {
	return c.get(name)
}

func (c MeshWorkspaceGroupBindingClient) Create(binding *MeshWorkspaceGroupBinding) (*MeshWorkspaceGroupBinding, error) {
	return c.post(binding)
}

func (c MeshWorkspaceGroupBindingClient) Delete(name string) error {
	return c.delete(name)
}
