package client

type MeshWorkspaceUserBinding struct {
	MeshWorkspaceBinding
}

type MeshWorkspaceUserBindingClient struct {
	meshObjectClient[MeshWorkspaceUserBinding]
}

func newWorkspaceUserBindingClient(c *httpClient) MeshWorkspaceUserBindingClient {
	return MeshWorkspaceUserBindingClient{newMeshObjectClient[MeshWorkspaceUserBinding](c, "v2", "meshworkspacebindings", "userbindings")}
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
