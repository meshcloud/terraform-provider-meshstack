package client

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
)

const CONTENT_TYPE_TENANT_V4 = "application/vnd.meshcloud.api.meshtenant.v4.hal+json"

type MeshTenantV4 struct {
	ApiVersion string               `json:"apiVersion" tfsdk:"api_version"`
	Kind       string               `json:"kind" tfsdk:"kind"`
	Metadata   MeshTenantMetadataV4 `json:"metadata" tfsdk:"metadata"`
	Spec       MeshTenantSpecV4     `json:"spec" tfsdk:"spec"`
	Status     MeshTenantStatusV4   `json:"status" tfsdk:"status"`
}

type MeshTenantMetadataV4 struct {
	UUID             string  `json:"uuid" tfsdk:"uuid"`
	OwnedByProject   string  `json:"ownedByProject" tfsdk:"owned_by_project"`
	OwnedByWorkspace string  `json:"ownedByWorkspace" tfsdk:"owned_by_workspace"`
	DeletedOn        *string `json:"deletedOn" tfsdk:"deleted_on"`
	CreatedOn        *string `json:"createdOn" tfsdk:"created_on"`
}

type MeshTenantSpecV4 struct {
	PlatformIdentifier    string            `json:"platformIdentifier" tfsdk:"platform_identifier"`
	LocalId               *string           `json:"localId" tfsdk:"local_id"`
	LandingZoneIdentifier string            `json:"landingZoneIdentifier" tfsdk:"landing_zone_identifier"`
	Quotas                []MeshTenantQuota `json:"quotas" tfsdk:"quotas"`
}

type MeshTenantStatusV4 struct {
	Tags                     map[string][]string `json:"tags" tfsdk:"tags"`
	LastReplicated           *string             `json:"lastReplicated" tfsdk:"last_replicated"`
	CurrentReplicationStatus string              `json:"currentReplicationStatus" tfsdk:"current_replication_status"`
}

type MeshTenantCreateV4 struct {
	Metadata MeshTenantCreateMetadataV4 `json:"metadata" tfsdk:"metadata"`
	Spec     MeshTenantCreateSpecV4     `json:"spec" tfsdk:"spec"`
}

type MeshTenantCreateMetadataV4 struct {
	UUID             string `json:"uuid" tfsdk:"uuid"`
	OwnedByProject   string `json:"ownedByProject" tfsdk:"owned_by_project"`
	OwnedByWorkspace string `json:"ownedByWorkspace" tfsdk:"owned_by_workspace"`
}

type MeshTenantCreateSpecV4 struct {
	PlatformIdentifier    string            `json:"platformIdentifier" tfsdk:"platform_identifier"`
	LocalId               *string           `json:"localId" tfsdk:"local_id"`
	LandingZoneIdentifier string            `json:"landingZoneIdentifier" tfsdk:"landing_zone_identifier"`
	Quotas                []MeshTenantQuota `json:"quotas" tfsdk:"quotas"`
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

func (c *MeshStackProviderClient) CreateTenantV4(tenant *MeshTenantCreateV4) (*MeshTenantV4, error) {
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

	if res.StatusCode != 200 {
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
