package client

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
)

const CONTENT_TYPE_PLATFORM = "application/vnd.meshcloud.api.meshplatform.v1.hal+json"

type MeshPlatform struct {
	ApiVersion string               `json:"apiVersion" tfsdk:"api_version"`
	Kind       string               `json:"kind" tfsdk:"kind"`
	Metadata   MeshPlatformMetadata `json:"metadata" tfsdk:"metadata"`
	Spec       MeshPlatformSpec     `json:"spec" tfsdk:"spec"`
}

type MeshPlatformMetadata struct {
	Name      string  `json:"name" tfsdk:"name"`
	CreatedOn string  `json:"createdOn" tfsdk:"created_on"`
	DeletedOn *string `json:"deletedOn" tfsdk:"deleted_on"`
}

type MeshPlatformSpec struct {
	DisplayName  string                 `json:"displayName" tfsdk:"display_name"`
	PlatformType string                 `json:"platformType" tfsdk:"platform_type"`
	Config       map[string]interface{} `json:"config,omitempty" tfsdk:"config"`
	Tags         map[string][]string    `json:"tags,omitempty" tfsdk:"tags"`
}

type MeshPlatformCreate struct {
	Metadata MeshPlatformCreateMetadata `json:"metadata" tfsdk:"metadata"`
	Spec     MeshPlatformSpec           `json:"spec" tfsdk:"spec"`
}

type MeshPlatformCreateMetadata struct {
	Name string `json:"name" tfsdk:"name"`
}

func (c *MeshStackProviderClient) urlForPlatform(name string) *url.URL {
	return c.endpoints.Platforms.JoinPath(name)
}

func (c *MeshStackProviderClient) ReadPlatform(name string) (*MeshPlatform, error) {
	targetUrl := c.urlForPlatform(name)
	req, err := http.NewRequest("GET", targetUrl.String(), nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", CONTENT_TYPE_PLATFORM)

	res, err := c.doAuthenticatedRequest(req)
	if err != nil {
		return nil, err
	}

	defer res.Body.Close()

	if res.StatusCode == http.StatusNotFound {
		return nil, nil // Not found is not an error
	}

	data, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}

	if !isSuccessHTTPStatus(res) {
		return nil, fmt.Errorf("unexpected status code: %d, %s", res.StatusCode, data)
	}

	var platform MeshPlatform
	err = json.Unmarshal(data, &platform)
	if err != nil {
		return nil, err
	}
	return &platform, nil
}

func (c *MeshStackProviderClient) CreatePlatform(platform *MeshPlatformCreate) (*MeshPlatform, error) {
	payload, err := json.Marshal(platform)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest("POST", c.endpoints.Platforms.String(), bytes.NewBuffer(payload))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", CONTENT_TYPE_PLATFORM)
	req.Header.Set("Accept", CONTENT_TYPE_PLATFORM)

	res, err := c.doAuthenticatedRequest(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	data, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}

	if !isSuccessHTTPStatus(res) {
		return nil, fmt.Errorf("unexpected status code: %d, %s", res.StatusCode, data)
	}

	var createdPlatform MeshPlatform
	err = json.Unmarshal(data, &createdPlatform)
	if err != nil {
		return nil, err
	}
	return &createdPlatform, nil
}

func (c *MeshStackProviderClient) UpdatePlatform(name string, platform *MeshPlatformCreate) (*MeshPlatform, error) {
	targetUrl := c.urlForPlatform(name)

	payload, err := json.Marshal(platform)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest("PUT", targetUrl.String(), bytes.NewBuffer(payload))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", CONTENT_TYPE_PLATFORM)
	req.Header.Set("Accept", CONTENT_TYPE_PLATFORM)

	res, err := c.doAuthenticatedRequest(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	data, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}

	if !isSuccessHTTPStatus(res) {
		return nil, fmt.Errorf("unexpected status code: %d, %s", res.StatusCode, data)
	}

	var updatedPlatform MeshPlatform
	err = json.Unmarshal(data, &updatedPlatform)
	if err != nil {
		return nil, err
	}
	return &updatedPlatform, nil
}

func (c *MeshStackProviderClient) DeletePlatform(name string) error {
	targetUrl := c.urlForPlatform(name)
	return c.deleteMeshObject(*targetUrl, 204)
}
