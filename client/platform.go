package client

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
)

const CONTENT_TYPE_PLATFORM = "application/vnd.meshcloud.api.meshplatform.v2-preview.hal+json"

type MeshPlatform struct {
	ApiVersion string               `json:"apiVersion" tfsdk:"api_version"`
	Kind       string               `json:"kind" tfsdk:"kind"`
	Metadata   MeshPlatformMetadata `json:"metadata" tfsdk:"metadata"`
	Spec       MeshPlatformSpec     `json:"spec" tfsdk:"spec"`
}

type MeshPlatformMetadata struct {
	Name             string  `json:"name" tfsdk:"name"`
	OwnedByWorkspace string  `json:"ownedByWorkspace" tfsdk:"owned_by_workspace"`
	Uuid             string  `json:"uuid" tfsdk:"uuid"`
	CreatedOn        string  `json:"createdOn" tfsdk:"created_on"`
	DeletedOn        *string `json:"deletedOn" tfsdk:"deleted_on"`
}

type MeshPlatformSpec struct {
	DisplayName            string               `json:"displayName" tfsdk:"display_name"`
	Description            string               `json:"description" tfsdk:"description"`
	Endpoint               string               `json:"endpoint" tfsdk:"endpoint"`
	SupportUrl             *string              `json:"supportUrl,omitempty" tfsdk:"support_url"`
	DocumentationUrl       *string              `json:"documentationUrl,omitempty" tfsdk:"documentation_url"`
	LocationRef            LocationRef          `json:"locationRef" tfsdk:"location_ref"`
	ContributingWorkspaces []string             `json:"contributingWorkspaces" tfsdk:"contributing_workspaces"`
	Availability           PlatformAvailability `json:"availability" tfsdk:"availability"`
	Config                 PlatformConfig       `json:"config" tfsdk:"config"`
	QuotaDefinitions       []QuotaDefinition    `json:"quotaDefinitions" tfsdk:"quota_definitions"`
}

type SecretEmbedded struct {
	Plaintext *string `json:"plaintext,omitempty" tfsdk:"plaintext"`
	// TODO: add Hash field
}

type QuotaDefinition struct {
	QuotaKey              string `json:"quotaKey" tfsdk:"quota_key"`
	MinValue              int    `json:"minValue" tfsdk:"min_value"`
	MaxValue              int    `json:"maxValue" tfsdk:"max_value"`
	Unit                  string `json:"unit" tfsdk:"unit"`
	AutoApprovalThreshold int    `json:"autoApprovalThreshold" tfsdk:"auto_approval_threshold"`
	Description           string `json:"description" tfsdk:"description"`
	Label                 string `json:"label" tfsdk:"label"`
}

type LocationRef struct {
	Kind string `json:"kind" tfsdk:"kind"`
	Name string `json:"name" tfsdk:"name"`
}

type PlatformAvailability struct {
	Restriction            string   `json:"restriction" tfsdk:"restriction"`
	PublicationState       string   `json:"publicationState" tfsdk:"publication_state"`
	RestrictedToWorkspaces []string `json:"restrictedToWorkspaces,omitempty" tfsdk:"restricted_to_workspaces"`
}

type PlatformConfig struct {
	Type       string                    `json:"type" tfsdk:"type"`
	Aws        *AwsPlatformConfig        `json:"aws,omitempty" tfsdk:"aws"`
	Aks        *AksPlatformConfig        `json:"aks,omitempty" tfsdk:"aks"`
	Azure      *AzurePlatformConfig      `json:"azure,omitempty" tfsdk:"azure"`
	AzureRg    *AzureRgPlatformConfig    `json:"azurerg,omitempty" tfsdk:"azurerg"`
	Gcp        *GcpPlatformConfig        `json:"gcp,omitempty" tfsdk:"gcp"`
	Kubernetes *KubernetesPlatformConfig `json:"kubernetes,omitempty" tfsdk:"kubernetes"`
	OpenShift  *OpenShiftPlatformConfig  `json:"openshift,omitempty" tfsdk:"openshift"`
}

type MeshPlatformCreate struct {
	ApiVersion string                     `json:"apiVersion" tfsdk:"api_version"`
	Metadata   MeshPlatformCreateMetadata `json:"metadata" tfsdk:"metadata"`
	Spec       MeshPlatformSpec           `json:"spec" tfsdk:"spec"`
}

type MeshPlatformCreateMetadata struct {
	Name             string `json:"name" tfsdk:"name"`
	OwnedByWorkspace string `json:"ownedByWorkspace" tfsdk:"owned_by_workspace"`
}

type MeshPlatformUpdate struct {
	ApiVersion string                     `json:"apiVersion" tfsdk:"api_version"`
	Metadata   MeshPlatformUpdateMetadata `json:"metadata" tfsdk:"metadata"`
	Spec       MeshPlatformSpec           `json:"spec" tfsdk:"spec"`
}

type MeshPlatformUpdateMetadata struct {
	Name             string `json:"name" tfsdk:"name"`
	OwnedByWorkspace string `json:"ownedByWorkspace" tfsdk:"owned_by_workspace"`
	Uuid             string `json:"uuid" tfsdk:"uuid"`
}

type MeshPlatformMeteringProcessingConfig struct {
	CompactTimelinesAfterDays int64 `json:"compactTimelinesAfterDays" tfsdk:"compact_timelines_after_days"`
	DeleteRawDataAfterDays    int64 `json:"deleteRawDataAfterDays" tfsdk:"delete_raw_data_after_days"`
}

type MeshTenantTags struct {
	NamespacePrefix string      `json:"namespacePrefix" tfsdk:"namespace_prefix"`
	TagMappers      []TagMapper `json:"tagMappers" tfsdk:"tag_mappers"`
}

type TagMapper struct {
	Key          string `json:"key" tfsdk:"key"`
	ValuePattern string `json:"valuePattern" tfsdk:"value_pattern"`
}

func (c *MeshStackProviderClient) urlForPlatform(uuid string) *url.URL {
	return c.endpoints.Platforms.JoinPath(uuid)
}

func (c *MeshStackProviderClient) ReadPlatform(uuid string) (*MeshPlatform, error) {
	targetUrl := c.urlForPlatform(uuid)
	req, err := http.NewRequest("GET", targetUrl.String(), nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", CONTENT_TYPE_PLATFORM)

	res, err := c.doAuthenticatedRequest(req)
	if err != nil {
		return nil, err
	}

	defer func() { _ = res.Body.Close() }()

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

	var createdPlatform MeshPlatform
	err = json.Unmarshal(data, &createdPlatform)
	if err != nil {
		return nil, err
	}
	return &createdPlatform, nil
}

func (c *MeshStackProviderClient) DeletePlatform(uuid string) error {
	targetUrl := c.urlForPlatform(uuid)
	return c.deleteMeshObject(*targetUrl, 204)
}

func (c *MeshStackProviderClient) UpdatePlatform(uuid string, platform *MeshPlatformUpdate) (*MeshPlatform, error) {
	targetUrl := c.urlForPlatform(uuid)

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

	var updatedPlatform MeshPlatform
	err = json.Unmarshal(data, &updatedPlatform)
	if err != nil {
		return nil, err
	}
	return &updatedPlatform, nil
}
