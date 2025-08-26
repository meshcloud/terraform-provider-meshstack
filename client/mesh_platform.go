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
	DisplayName  string              `json:"displayName" tfsdk:"display_name"`
	PlatformType string              `json:"platformType" tfsdk:"platform_type"`
	Description  *string             `json:"description,omitempty" tfsdk:"description"`
	Tags         map[string][]string `json:"tags,omitempty" tfsdk:"tags"`
	Config       *PlatformConfig     `json:"config,omitempty" tfsdk:"config"`
}

// PlatformConfig holds configuration for different platform types
type PlatformConfig struct {
	AWS *AWSPlatformConfig `json:"aws,omitempty" tfsdk:"aws"`
	// Future platform types can be added here:
	// Azure    *AzurePlatformConfig    `json:"azure,omitempty" tfsdk:"azure"`
	// OpenStack *OpenStackPlatformConfig `json:"openstack,omitempty" tfsdk:"openstack"`
}

// AWSPlatformConfig represents AWS platform configuration
// Based on the meshStack API documentation for AWS platforms
type AWSPlatformConfig struct {
	// AWS Account ID
	AccountId string `json:"accountId" tfsdk:"account_id"`
	
	// AWS Region  
	Region string `json:"region" tfsdk:"region"`
	
	// AWS API endpoint URL (optional, defaults to standard AWS endpoints)
	EndpointUrl *string `json:"endpointUrl,omitempty" tfsdk:"endpoint_url"`
	
	// IAM Role ARN for cross-account access (optional)
	RoleArn *string `json:"roleArn,omitempty" tfsdk:"role_arn"`
	
	// External ID for role assumption (optional, used with RoleArn)
	ExternalId *string `json:"externalId,omitempty" tfsdk:"external_id"`
	
	// Additional AWS-specific configuration options
	AssumeRoleSessionName *string `json:"assumeRoleSessionName,omitempty" tfsdk:"assume_role_session_name"`
}

type MeshPlatformCreate struct {
	ApiVersion string                     `json:"apiVersion" tfsdk:"api_version"`
	Metadata   MeshPlatformCreateMetadata `json:"metadata" tfsdk:"metadata"`
	Spec       MeshPlatformSpec           `json:"spec" tfsdk:"spec"`
}

type MeshPlatformCreateMetadata struct {
	Name string `json:"name" tfsdk:"name"`
}

func (c *MeshStackProviderClient) urlForPlatform(identifier string) *url.URL {
	return c.endpoints.Platforms.JoinPath(identifier)
}

func (c *MeshStackProviderClient) ReadPlatform(identifier string) (*MeshPlatform, error) {
	targetUrl := c.urlForPlatform(identifier)
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

func (c *MeshStackProviderClient) UpdatePlatform(identifier string, platform *MeshPlatformCreate) (*MeshPlatform, error) {
	targetUrl := c.urlForPlatform(identifier)

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

func (c *MeshStackProviderClient) DeletePlatform(identifier string) error {
	targetUrl := c.urlForPlatform(identifier)
	return c.deleteMeshObject(*targetUrl, 204)
}
