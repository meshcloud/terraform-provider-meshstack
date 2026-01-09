package client

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
	meshObjectClient[MeshTenant]
}

func newTenantClient(c *httpClient) MeshTenantClient {
	return MeshTenantClient{newMeshObjectClient[MeshTenant](c, "v3")}
}

func (c MeshTenantClient) tenantId(workspace string, project string, platform string) string {
	return workspace + "." + project + "." + platform
}

func (c MeshTenantClient) Read(workspace string, project string, platform string) (*MeshTenant, error) {
	return c.get(c.tenantId(workspace, project, platform))
}

func (c MeshTenantClient) Create(tenant *MeshTenantCreate) (*MeshTenant, error) {
	return c.post(tenant)
}

func (c MeshTenantClient) Delete(workspace string, project string, platform string) error {
	return c.delete(c.tenantId(workspace, project, platform))
}
