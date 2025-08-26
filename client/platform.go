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
	Name             string  `json:"name" tfsdk:"name"`
	OwnedByWorkspace string  `json:"ownedByWorkspace" tfsdk:"owned_by_workspace"`
	CreatedOn        string  `json:"createdOn" tfsdk:"created_on"`
	DeletedOn        *string `json:"deletedOn" tfsdk:"deleted_on"`
}

type MeshPlatformSpec struct {
	DisplayName            string                    `json:"displayName" tfsdk:"display_name"`
	LocationRef            *LocationRef              `json:"locationRef,omitempty" tfsdk:"location_ref"`
	Description            *string                   `json:"description,omitempty" tfsdk:"description"`
	Endpoint               string                    `json:"endpoint" tfsdk:"endpoint"`
	SupportUrl             *string                   `json:"supportUrl,omitempty" tfsdk:"support_url"`
	DocumentationUrl       *string                   `json:"documentationUrl,omitempty" tfsdk:"documentation_url"`
	Availability           *MeshPlatformAvailability `json:"availability,omitempty" tfsdk:"availability"`
	Config                 *PlatformConfig           `json:"config,omitempty" tfsdk:"config"`
	ContributingWorkspaces []string                  `json:"contributingWorkspaces,omitempty" tfsdk:"contributing_workspaces"`
}

type MeshPlatformAvailability struct {
	Restriction            string   `json:"restriction" tfsdk:"restriction"`
	RestrictedToWorkspaces []string `json:"restrictedToWorkspaces,omitempty" tfsdk:"restricted_to_workspaces"`
	MarketplaceStatus      string   `json:"marketplaceStatus" tfsdk:"marketplace_status"`
}

type LocationRef struct {
	Kind       string `json:"kind" tfsdk:"kind"`
	Identifier string `json:"identifier" tfsdk:"identifier"`
}

// PlatformConfig holds configuration for different platform types
type PlatformConfig struct {
	Type       string                    `json:"type" tfsdk:"type"`
	AWS        *AWSPlatformConfig        `json:"aws,omitempty" tfsdk:"aws"`
	AKS        *AKSPlatformConfig        `json:"aks,omitempty" tfsdk:"aks"`
	Azure      *AzurePlatformConfig      `json:"azure,omitempty" tfsdk:"azure"`
	AzureRG    *AzureRGPlatformConfig    `json:"azurerg,omitempty" tfsdk:"azurerg"`
	GCP        *GCPPlatformConfig        `json:"gcp,omitempty" tfsdk:"gcp"`
	Kubernetes *KubernetesPlatformConfig `json:"kubernetes,omitempty" tfsdk:"kubernetes"`
	OpenShift  *OpenShiftPlatformConfig  `json:"openshift,omitempty" tfsdk:"openshift"`
}

type PlatformCreate struct {
	ApiVersion string                 `json:"apiVersion" tfsdk:"api_version"`
	Metadata   PlatformCreateMetadata `json:"metadata" tfsdk:"metadata"`
	Spec       MeshPlatformSpec       `json:"spec" tfsdk:"spec"`
}

type PlatformCreateMetadata struct {
	Name             string `json:"name" tfsdk:"name"`
	OwnedByWorkspace string `json:"ownedByWorkspace" tfsdk:"owned_by_workspace"`
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

func (c *MeshStackProviderClient) CreatePlatform(platform *PlatformCreate) (*MeshPlatform, error) {
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

func (c *MeshStackProviderClient) UpdatePlatform(identifier string, platform *PlatformCreate) (*MeshPlatform, error) {
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
