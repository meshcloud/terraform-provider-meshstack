package testconfig

import "testing"

// TenantV4 builds a tenant_v4 config using the provided prerequisite addresses.
func TenantV4(t *testing.T, projectAddr, platformAddr, landingZoneAddr Traversal) (config Config, tenantAddr Traversal) {
	t.Helper()
	return Resource{Name: "tenant_v4"}.Config(t).WithFirstBlock(
		ExtractAddress(&tenantAddr),
		Descend("metadata")(
			Descend("owned_by_workspace")(SetAddr(projectAddr, "metadata", "owned_by_workspace")),
			Descend("owned_by_project")(SetAddr(projectAddr, "metadata", "name")),
		),
		Descend("spec")(
			Descend("platform_identifier")(SetAddr(platformAddr, "identifier")),
			Descend("landing_zone_identifier")(SetAddr(landingZoneAddr, "metadata", "name")),
		),
	), tenantAddr
}

// TenantV4AndWorkspace builds a tenant v4 config with all prerequisites (workspace, project, platform, landing zone).
func TenantV4AndWorkspace(t *testing.T) (config Config, tenantAddr Traversal) {
	t.Helper()
	workspaceConfig, workspaceAddr := Workspace(t)
	projectConfig, projectAddr := Project(t, workspaceAddr)
	platformConfig, platformAddr, platformTypeAddr := CustomPlatform(t, workspaceAddr)
	landingZoneConfig, landingZoneAddr := LandingZone(t, workspaceAddr, platformAddr, platformTypeAddr)

	tenantConfig, tenantAddr := TenantV4(t, projectAddr, platformAddr, landingZoneAddr)
	return tenantConfig.Join(workspaceConfig, projectConfig, platformConfig, landingZoneConfig), tenantAddr
}
