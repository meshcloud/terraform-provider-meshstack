package testconfig

import "testing"

func BuildTenantConfig(t *testing.T) (config Config, tenantAddr Traversal) {
	t.Helper()
	workspaceConfig, workspaceAddr := BuildWorkspaceConfig(t)
	projectConfig, projectAddr := BuildProjectConfig(t, workspaceAddr)
	platformConfig, platformAddr, platformTypeAddr := BuildCustomPlatformConfig(t, workspaceAddr)
	landingZoneConfig, landingZoneAddr := BuildLandingZoneConfig(t, workspaceAddr, platformAddr, platformTypeAddr)

	return Resource{Name: "tenant_v4"}.Config(t).WithFirstBlock(t,
		ExtractIdentifier(&tenantAddr),
		Traverse(t, "metadata")(
			Traverse(t, "owned_by_workspace")(SetRawExpr(projectAddr.Join("metadata", "owned_by_workspace").String())),
			Traverse(t, "owned_by_project")(SetRawExpr(projectAddr.Join("metadata", "name").String())),
		),
		Traverse(t, "spec")(
			Traverse(t, "platform_identifier")(SetRawExpr(platformAddr.Join("identifier").String())),
			Traverse(t, "landing_zone_identifier")(SetRawExpr(landingZoneAddr.Join("metadata", "name").String())),
		),
	).Join(workspaceConfig, projectConfig, platformConfig, landingZoneConfig), tenantAddr
}
