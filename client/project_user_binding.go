package client

import (
	"net/url"
)

const CONTENT_TYPE_PROJECT_USER_BINDING = "application/vnd.meshcloud.api.meshprojectuserbinding.v3.hal+json"

type MeshProjectUserBinding = MeshProjectBinding

func (c *MeshStackProviderClient) urlForPojectUserBinding(name string) *url.URL {
	return c.endpoints.ProjectUserBindings.JoinPath(name)
}

func (c *MeshStackProviderClient) ReadProjectUserBinding(name string) (*MeshProjectUserBinding, error) {
	return unmarshalBodyIfPresent[MeshProjectBinding](c.doAuthenticatedRequest("GET", c.urlForPojectUserBinding(name),
		withAccept(CONTENT_TYPE_PROJECT_USER_BINDING),
	))
}

func (c *MeshStackProviderClient) CreateProjectUserBinding(binding *MeshProjectUserBinding) (*MeshProjectUserBinding, error) {
	return unmarshalBody[MeshProjectBinding](c.doAuthenticatedRequest("POST", c.endpoints.ProjectUserBindings,
		withPayload(binding, CONTENT_TYPE_PROJECT_USER_BINDING),
	))
}

func (c *MeshStackProviderClient) DeleteProjecUserBinding(name string) error {
	_, err := c.doAuthenticatedRequest("DELETE", c.urlForPojectUserBinding(name))
	return err
}
