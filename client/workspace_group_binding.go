package client

import (
	"net/url"
)

const CONTENT_TYPE_WORKSPACE_GROUP_BINDING = "application/vnd.meshcloud.api.meshworkspacegroupbinding.v2.hal+json"

type MeshWorkspaceGroupBinding = MeshWorkspaceBinding

func (c *MeshStackProviderClient) urlForWorkspaceGroupBinding(name string) *url.URL {
	return c.endpoints.WorkspaceGroupBindings.JoinPath(name)
}

func (c *MeshStackProviderClient) ReadWorkspaceGroupBinding(name string) (*MeshWorkspaceGroupBinding, error) {
	return c.readWorkspaceBinding(name, CONTENT_TYPE_WORKSPACE_GROUP_BINDING)
}

func (c *MeshStackProviderClient) CreateWorkspaceGroupBinding(binding *MeshWorkspaceGroupBinding) (*MeshWorkspaceGroupBinding, error) {
	return c.createWorkspaceBinding(binding, CONTENT_TYPE_WORKSPACE_GROUP_BINDING)
}

func (c *MeshStackProviderClient) DeleteWorkspaceGroupBinding(name string) error {
	targetUrl := c.urlForWorkspaceGroupBinding(name)
	return c.deleteMeshObject(*targetUrl, 204)
}
