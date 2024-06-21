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
	return c.readProjectBinding(name, CONTENT_TYPE_PROJECT_USER_BINDING)
}

func (c *MeshStackProviderClient) CreateProjectUserBinding(binding *MeshProjectUserBinding) (*MeshProjectUserBinding, error) {
	return c.createProjectBinding(binding, CONTENT_TYPE_PROJECT_USER_BINDING)
}

func (c *MeshStackProviderClient) DeleteProjecUserBinding(name string) error {
	targetUrl := c.urlForPojectUserBinding(name)
	return c.deleteMeshObject(*targetUrl, 204)
}
