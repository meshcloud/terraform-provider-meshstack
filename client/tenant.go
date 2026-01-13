package client

import (
	"context"

	"github.com/meshcloud/terraform-provider-meshstack/client/internal"
)

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

type MeshTenantClient struct {
	meshObject internal.MeshObjectClient[MeshTenant]
}

func newTenantClient(ctx context.Context, httpClient *internal.HttpClient) MeshTenantClient {
	return MeshTenantClient{internal.NewMeshObjectClient[MeshTenant](ctx, httpClient, "v3")}
}

func (c MeshTenantClient) tenantId(workspace string, project string, platform string) string {
	return workspace + "." + project + "." + platform
}

func (c MeshTenantClient) Read(ctx context.Context, workspace string, project string, platform string) (*MeshTenant, error) {
	return c.meshObject.Get(ctx, c.tenantId(workspace, project, platform))
}

func (c MeshTenantClient) Create(ctx context.Context, tenant *MeshTenantCreate) (*MeshTenant, error) {
	return c.meshObject.Post(ctx, tenant)
}

func (c MeshTenantClient) Delete(ctx context.Context, workspace string, project string, platform string) error {
	return c.meshObject.Delete(ctx, c.tenantId(workspace, project, platform))
}
