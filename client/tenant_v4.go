package client

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
)

const CONTENT_TYPE_TENANT_V4 = "application/vnd.meshcloud.api.meshtenant.v4-preview.hal+json"

type MeshTenantV4 struct {
	ApiVersion string               `json:"apiVersion" tfsdk:"api_version"`
	Kind       string               `json:"kind" tfsdk:"kind"`
	Metadata   MeshTenantV4Metadata `json:"metadata" tfsdk:"metadata"`
	Spec       MeshTenantV4Spec     `json:"spec" tfsdk:"spec"`
	Status     MeshTenantV4Status   `json:"status" tfsdk:"status"`
}

type MeshTenantV4Metadata struct {
	Uuid                string  `json:"uuid" tfsdk:"uuid"`
	OwnedByProject      string  `json:"ownedByProject" tfsdk:"owned_by_project"`
	OwnedByWorkspace    string  `json:"ownedByWorkspace" tfsdk:"owned_by_workspace"`
	CreatedOn           string  `json:"createdOn" tfsdk:"created_on"`
	MarkedForDeletionOn *string `json:"markedForDeletionOn" tfsdk:"marked_for_deletion_on"`
	DeletedOn           *string `json:"deletedOn" tfsdk:"deleted_on"`
}

type MeshTenantV4Spec struct {
	PlatformIdentifier    string             `json:"platformIdentifier" tfsdk:"platform_identifier"`
	PlatformTenantId      *string            `json:"platformTenantId" tfsdk:"platform_tenant_id"`
	LandingZoneIdentifier *string            `json:"landingZoneIdentifier" tfsdk:"landing_zone_identifier"`
	Quotas                *[]MeshTenantQuota `json:"quotas" tfsdk:"quotas"`
}

type MeshTenantV4Status struct {
	TenantName                  string              `json:"tenantName" tfsdk:"tenant_name"`
	PlatformTypeIdentifier      string              `json:"platformTypeIdentifier" tfsdk:"platform_type_identifier"`
	PlatformWorkspaceIdentifier *string             `json:"platformWorkspaceIdentifier" tfsdk:"platform_workspace_identifier"`
	Tags                        map[string][]string `json:"tags" tfsdk:"tags"`
}

type MeshTenantV4Create struct {
	Metadata MeshTenantV4CreateMetadata `json:"metadata" tfsdk:"metadata"`
	Spec     MeshTenantV4CreateSpec     `json:"spec" tfsdk:"spec"`
}

type MeshTenantV4CreateMetadata struct {
	OwnedByProject   string `json:"ownedByProject" tfsdk:"owned_by_project"`
	OwnedByWorkspace string `json:"ownedByWorkspace" tfsdk:"owned_by_workspace"`
}

type MeshTenantV4CreateSpec struct {
	PlatformIdentifier    string             `json:"platformIdentifier" tfsdk:"platform_identifier"`
	LandingZoneIdentifier *string            `json:"landingZoneIdentifier" tfsdk:"landing_zone_identifier"`
	PlatformTenantId      *string            `json:"platformTenantId" tfsdk:"platform_tenant_id"`
	Quotas                *[]MeshTenantQuota `json:"quotas" tfsdk:"quotas"`
}

func (c *MeshStackProviderClient) urlForTenantV4(uuid string) *url.URL {
	return c.endpoints.Tenants.JoinPath(uuid)
}

func (c *MeshStackProviderClient) ReadTenantV4(uuid string) (*MeshTenantV4, error) {
	targetUrl := c.urlForTenantV4(uuid)
	req, err := http.NewRequest("GET", targetUrl.String(), nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", CONTENT_TYPE_TENANT_V4)

	res, err := c.doAuthenticatedRequest(req)
	if err != nil {
		return nil, err
	}

	defer res.Body.Close()

	data, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}

	if res.StatusCode == 404 {
		return nil, nil
	}

	if res.StatusCode != 200 {
		return nil, fmt.Errorf("unexpected status code: %d, %s", res.StatusCode, data)
	}

	var tenant MeshTenantV4
	err = json.Unmarshal(data, &tenant)
	if err != nil {
		return nil, err
	}

	return &tenant, nil
}

func (c *MeshStackProviderClient) CreateTenantV4(tenant *MeshTenantV4Create) (*MeshTenantV4, error) {
	payload, err := json.Marshal(tenant)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest("POST", c.endpoints.Tenants.String(), bytes.NewBuffer(payload))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", CONTENT_TYPE_TENANT_V4)
	req.Header.Set("Accept", CONTENT_TYPE_TENANT_V4)

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

	var createdTenant MeshTenantV4
	err = json.Unmarshal(data, &createdTenant)
	if err != nil {
		return nil, err
	}

	return &createdTenant, nil
}

func (c *MeshStackProviderClient) DeleteTenantV4(uuid string) error {
	targetUrl := c.urlForTenantV4(uuid)
	return c.deleteMeshObject(*targetUrl, 202)
}
