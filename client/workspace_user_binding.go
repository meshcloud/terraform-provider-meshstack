package client

type MeshWorkspaceUserBinding = MeshWorkspaceBinding

type MeshWorkspaceUserBindingClient struct {
	meshObjectClient[MeshWorkspaceBinding]
}

func (c MeshWorkspaceUserBindingClient) Read(name string) (*MeshWorkspaceUserBinding, error) {
	return c.get(name)
}

func (c MeshWorkspaceUserBindingClient) Create(binding *MeshWorkspaceUserBinding) (*MeshWorkspaceUserBinding, error) {
	return c.post(binding)
}

func (c MeshWorkspaceUserBindingClient) Delete(name string) error {
	return c.delete(name)
}
