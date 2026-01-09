package client

type MeshWorkspaceGroupBinding struct {
	MeshWorkspaceBinding
}

type MeshWorkspaceGroupBindingClient struct {
	meshObjectClient[MeshWorkspaceGroupBinding]
}

func newWorkspaceGroupBindingClient(c *httpClient) MeshWorkspaceGroupBindingClient {
	return MeshWorkspaceGroupBindingClient{newMeshObjectClient[MeshWorkspaceGroupBinding](c, "v2", "meshworkspacebindings", "groupbindings")}
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
