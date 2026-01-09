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
	return unmarshalBodyIfPresent[MeshWorkspaceBinding](c.doAuthenticatedRequest("GET", c.urlForWorkspaceGroupBinding(name),
		withAccept(CONTENT_TYPE_WORKSPACE_GROUP_BINDING),
	))
}

func (c *MeshStackProviderClient) CreateWorkspaceGroupBinding(binding *MeshWorkspaceGroupBinding) (*MeshWorkspaceGroupBinding, error) {
	return unmarshalBody[MeshWorkspaceBinding](c.doAuthenticatedRequest("POST", c.endpoints.WorkspaceGroupBindings,
		withPayload(binding, CONTENT_TYPE_WORKSPACE_GROUP_BINDING),
	))
}

func (c *MeshStackProviderClient) DeleteWorkspaceGroupBinding(name string) error {
	_, err := c.doAuthenticatedRequest("DELETE", c.urlForWorkspaceGroupBinding(name))
	return err
}
