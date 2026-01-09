package client

import (
	"bytes"
	"encoding/json"
	"net/http"
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
	targetUrl := c.urlForLocation(name)

	req, err := http.NewRequest("GET", targetUrl.String(), nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", CONTENT_TYPE_LOCATION)

	return unmarshalBodyIfPresent[MeshLocation](c.doAuthenticatedRequest(req))
}

func (c *MeshStackProviderClient) CreateLocation(location *MeshLocationCreate) (*MeshLocation, error) {
	payload, err := json.Marshal(location)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest("POST", c.endpoints.Locations.String(), bytes.NewBuffer(payload))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", CONTENT_TYPE_LOCATION)
	req.Header.Set("Accept", CONTENT_TYPE_LOCATION)

	return unmarshalBody[MeshLocation](c.doAuthenticatedRequest(req))
}

func (c *MeshStackProviderClient) UpdateLocation(name string, location *MeshLocationCreate) (*MeshLocation, error) {
	targetUrl := c.urlForLocation(name)

	payload, err := json.Marshal(location)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest("PUT", targetUrl.String(), bytes.NewBuffer(payload))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", CONTENT_TYPE_LOCATION)
	req.Header.Set("Accept", CONTENT_TYPE_LOCATION)

	return unmarshalBody[MeshLocation](c.doAuthenticatedRequest(req))
}

func (c *MeshStackProviderClient) DeleteLocation(name string) error {
	targetUrl := c.urlForLocation(name)
	return c.deleteMeshObject(*targetUrl, 204)
}
