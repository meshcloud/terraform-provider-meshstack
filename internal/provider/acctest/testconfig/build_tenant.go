package testconfig

import "testing"

// Tenant builds a meshstack_tenant config (v4 body) using the provided prerequisite addresses.
// The example resource.tf illustrates resolving the refs from marketplace data sources; the builder
// keeps only the tenant block (FirstBlockOnly) and rewrites its refs to the given resource addresses.
func Tenant(t *testing.T, projectAddr, platformAddr, landingZoneAddr Traversal) (config Config, tenantAddr Traversal) {
	t.Helper()
	return Resource{Name: "tenant"}.Config(t).FirstBlockOnly().WithFirstBlock(
		ExtractAddress(&tenantAddr),
		Descend("metadata")(
			Descend("owned_by_workspace")(SetAddr(projectAddr, "metadata", "owned_by_workspace")),
			Descend("owned_by_project")(SetAddr(projectAddr, "metadata", "name")),
		),
		Descend("spec")(
			Descend("platform_ref")(SetRawExpr("%s", platformAddr.Join("ref"))),
			Descend("landing_zone_ref")(SetRawExpr("%s", landingZoneAddr.Join("ref"))),
		),
	), tenantAddr
}

// TenantAndWorkspace builds a meshstack_tenant config with all prerequisites (workspace, project, platform, landing zone).
func TenantAndWorkspace(t *testing.T) (config Config, tenantAddr Traversal) {
	t.Helper()
	workspaceConfig, workspaceAddr := Workspace(t)
	projectConfig, projectAddr := Project(t, workspaceAddr)
	platformConfig, platformAddr, platformTypeAddr := CustomPlatform(t, workspaceAddr)
	landingZoneConfig, landingZoneAddr := LandingZone(t, workspaceAddr, platformAddr, platformTypeAddr)

	tenantConfig, tenantAddr := Tenant(t, projectAddr, platformAddr, landingZoneAddr)
	return tenantConfig.Join(workspaceConfig, projectConfig, platformConfig, landingZoneConfig), tenantAddr
}
