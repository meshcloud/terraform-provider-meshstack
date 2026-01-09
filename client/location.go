package client

import (
	"net/url"
)

const CONTENT_TYPE_LOCATION = "application/vnd.meshcloud.api.meshlocation.v1-preview.hal+json"

type MeshLocation struct {
	ApiVersion string               `json:"apiVersion" tfsdk:"api_version"`
	Metadata   MeshLocationMetadata `json:"metadata" tfsdk:"metadata"`
	Spec       MeshLocationSpec     `json:"spec" tfsdk:"spec"`
	Status     MeshLocationStatus   `json:"status" tfsdk:"status"`
}

type MeshLocationMetadata struct {
	Name string `json:"name" tfsdk:"name"`
	Uuid string `json:"uuid" tfsdk:"uuid"`
}

type MeshLocationSpec struct {
	DisplayName string `json:"displayName" tfsdk:"display_name"`
	Description string `json:"description" tfsdk:"description"`
}

type MeshLocationStatus struct {
	IsPublic bool `json:"isPublic" tfsdk:"is_public"`
}

type MeshLocationCreate struct {
	ApiVersion string                     `json:"apiVersion" tfsdk:"api_version"`
	Metadata   MeshLocationCreateMetadata `json:"metadata" tfsdk:"metadata"`
	Spec       MeshLocationSpec           `json:"spec" tfsdk:"spec"`
}

type MeshLocationCreateMetadata struct {
	Name string `json:"name" tfsdk:"name"`
}

func (c *MeshStackProviderClient) urlForLocation(name string) *url.URL {
	return c.endpoints.Locations.JoinPath(name)
}

func (c *MeshStackProviderClient) ReadLocation(name string) (*MeshLocation, error) {
	return unmarshalBodyIfPresent[MeshLocation](c.doAuthenticatedRequest("GET", c.urlForLocation(name),
		withAccept(CONTENT_TYPE_LOCATION),
	))
}

func (c *MeshStackProviderClient) CreateLocation(location *MeshLocationCreate) (*MeshLocation, error) {
	return unmarshalBody[MeshLocation](c.doAuthenticatedRequest("POST", c.endpoints.Locations,
		withPayload(location, CONTENT_TYPE_LOCATION),
	))
}

func (c *MeshStackProviderClient) UpdateLocation(name string, location *MeshLocationCreate) (*MeshLocation, error) {
	return unmarshalBody[MeshLocation](c.doAuthenticatedRequest("PUT", c.urlForLocation(name),
		withPayload(location, CONTENT_TYPE_LOCATION),
	))
}

func (c *MeshStackProviderClient) DeleteLocation(name string) error {
	_, err := c.doAuthenticatedRequest("DELETE", c.urlForLocation(name))
	return err
}
