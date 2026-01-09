package client

import (
	"net/url"
)

const CONTENT_TYPE_PROJECT_GROUP_BINDING = "application/vnd.meshcloud.api.meshprojectgroupbinding.v3.hal+json"

type MeshProjectGroupBinding = MeshProjectBinding

func (c *MeshStackProviderClient) urlForPojectGroupBinding(name string) *url.URL {
	return c.endpoints.ProjectGroupBindings.JoinPath(name)
}

func (c *MeshStackProviderClient) ReadProjectGroupBinding(name string) (*MeshProjectGroupBinding, error) {
	return unmarshalBodyIfPresent[MeshProjectBinding](c.doAuthenticatedRequest("GET", c.urlForPojectGroupBinding(name),
		withAccept(CONTENT_TYPE_PROJECT_GROUP_BINDING),
	))
}

func (c *MeshStackProviderClient) CreateProjectGroupBinding(binding *MeshProjectGroupBinding) (*MeshProjectGroupBinding, error) {
	return unmarshalBody[MeshProjectBinding](c.doAuthenticatedRequest("POST", c.endpoints.ProjectGroupBindings,
		withPayload(binding, CONTENT_TYPE_PROJECT_GROUP_BINDING),
	))
}

func (c *MeshStackProviderClient) DeleteProjecGroupBinding(name string) error {
	_, err := c.doAuthenticatedRequest("DELETE", c.urlForPojectGroupBinding(name))
	return err
}
