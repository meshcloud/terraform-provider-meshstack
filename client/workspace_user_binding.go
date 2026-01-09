package client

import (
	"net/url"
)

const CONTENT_TYPE_WORKSPACE_USER_BINDING = "application/vnd.meshcloud.api.meshworkspaceuserbinding.v2.hal+json"

type MeshWorkspaceUserBinding = MeshWorkspaceBinding

func (c *MeshStackProviderClient) urlForWorkspaceUserBinding(name string) *url.URL {
	return c.endpoints.WorkspaceUserBindings.JoinPath(name)
}

func (c *MeshStackProviderClient) ReadWorkspaceUserBinding(name string) (*MeshWorkspaceUserBinding, error) {
	return c.readWorkspaceBinding(name, CONTENT_TYPE_WORKSPACE_USER_BINDING)
}

func (c *MeshStackProviderClient) CreateWorkspaceUserBinding(binding *MeshWorkspaceUserBinding) (*MeshWorkspaceUserBinding, error) {
	return c.createWorkspaceBinding(binding, CONTENT_TYPE_WORKSPACE_USER_BINDING)
}

func (c *MeshStackProviderClient) DeleteWorkspaceUserBinding(name string) error {
	_, err := c.doAuthenticatedRequest("DELETE", c.urlForWorkspaceUserBinding(name))
	return err
}
