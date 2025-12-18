package client

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
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

	res, err := c.doAuthenticatedRequest(req)
	if err != nil {
		return nil, err
	}

	defer func() {
		_ = res.Body.Close()
	}()

	if res.StatusCode == http.StatusNotFound {
		return nil, nil
	}

	data, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}

	if !isSuccessHTTPStatus(res) {
		return nil, fmt.Errorf("unexpected status code: %d, %s", res.StatusCode, data)
	}

	var location MeshLocation
	err = json.Unmarshal(data, &location)
	if err != nil {
		return nil, err
	}

	return &location, nil
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

	res, err := c.doAuthenticatedRequest(req)
	if err != nil {
		return nil, err
	}

	defer func() {
		_ = res.Body.Close()
	}()

	data, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}

	if !isSuccessHTTPStatus(res) {
		return nil, fmt.Errorf("unexpected status code: %d, %s", res.StatusCode, data)
	}

	var createdLocation MeshLocation
	err = json.Unmarshal(data, &createdLocation)
	if err != nil {
		return nil, err
	}

	return &createdLocation, nil
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

	res, err := c.doAuthenticatedRequest(req)
	if err != nil {
		return nil, err
	}

	defer func() {
		_ = res.Body.Close()
	}()

	data, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}

	if !isSuccessHTTPStatus(res) {
		return nil, fmt.Errorf("unexpected status code: %d, %s", res.StatusCode, data)
	}

	var updatedLocation MeshLocation
	err = json.Unmarshal(data, &updatedLocation)
	if err != nil {
		return nil, err
	}

	return &updatedLocation, nil
}

func (c *MeshStackProviderClient) DeleteLocation(name string) error {
	targetUrl := c.urlForLocation(name)
	return c.deleteMeshObject(*targetUrl, 204)
}
