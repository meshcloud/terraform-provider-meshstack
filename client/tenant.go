package client

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/url"
)

const CONTENT_TYPE_TENANT = "application/vnd.meshcloud.api.meshtenant.v3.hal+json"

type MeshTenant struct {
	ApiVersion string             `json:"apiVersion" tfsdk:"api_version"`
	Kind       string             `json:"kind" tfsdk:"kind"`
	Metadata   MeshTenantMetadata `json:"metadata" tfsdk:"metadata"`
	Spec       MeshTenantSpec     `json:"spec" tfsdk:"spec"`
}

type MeshTenantMetadata struct {
	OwnedByProject     string              `json:"ownedByProject" tfsdk:"owned_by_project"`
	OwnedByWorkspace   string              `json:"ownedByWorkspace" tfsdk:"owned_by_workspace"`
	PlatformIdentifier string              `json:"platformIdentifier" tfsdk:"platform_identifier"`
	AssignedTags       map[string][]string `json:"assignedTags" tfsdk:"assigned_tags"`
	DeletedOn          *string             `json:"deletedOn" tfsdk:"deleted_on"`
}

type MeshTenantSpec struct {
	LocalId               *string           `json:"localId" tfsdk:"local_id"`
	LandingZoneIdentifier string            `json:"landingZoneIdentifier" tfsdk:"landing_zone_identifier"`
	Quotas                []MeshTenantQuota `json:"quotas" tfsdk:"quotas"`
}

type MeshTenantQuota struct {
	Key   string `json:"key" tfsdk:"key"`
	Value int64  `json:"value" tfsdk:"value"`
}

type MeshTenantCreate struct {
	Metadata MeshTenantCreateMetadata `json:"metadata" tfsdk:"metadata"`
	Spec     MeshTenantCreateSpec     `json:"spec" tfsdk:"spec"`
}

type MeshTenantCreateMetadata struct {
	OwnedByProject     string `json:"ownedByProject" tfsdk:"owned_by_project"`
	OwnedByWorkspace   string `json:"ownedByWorkspace" tfsdk:"owned_by_workspace"`
	PlatformIdentifier string `json:"platformIdentifier" tfsdk:"platform_identifier"`
}

type MeshTenantCreateSpec struct {
	LocalId               *string            `json:"localId" tfsdk:"local_id"`
	LandingZoneIdentifier *string            `json:"landingZoneIdentifier" tfsdk:"landing_zone_identifier"`
	Quotas                *[]MeshTenantQuota `json:"quotas" tfsdk:"quotas"`
}

func (c *MeshStackProviderClient) urlForTenant(workspace string, project string, platform string) *url.URL {
	identifier := workspace + "." + project + "." + platform
	return c.endpoints.Tenants.JoinPath(identifier)
}

func (c *MeshStackProviderClient) ReadTenant(workspace string, project string, platform string) (*MeshTenant, error) {
	targetUrl := c.urlForTenant(workspace, project, platform)
	req, err := http.NewRequest("GET", targetUrl.String(), nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", CONTENT_TYPE_TENANT)

	return unmarshalBodyIfPresent[MeshTenant](c.doAuthenticatedRequest(req))
}

func (c *MeshStackProviderClient) CreateTenant(tenant *MeshTenantCreate) (*MeshTenant, error) {
	payload, err := json.Marshal(tenant)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest("POST", c.endpoints.Tenants.String(), bytes.NewBuffer(payload))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", CONTENT_TYPE_TENANT)
	req.Header.Set("Accept", CONTENT_TYPE_TENANT)

	return unmarshalBody[MeshTenant](c.doAuthenticatedRequest(req))
}

func (c *MeshStackProviderClient) DeleteTenant(workspace string, project string, platform string) error {
	targetUrl := c.urlForTenant(workspace, project, platform)
	return c.deleteMeshObject(*targetUrl, 202)
}
