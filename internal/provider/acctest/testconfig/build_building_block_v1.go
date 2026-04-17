package testconfig

import "testing"

// BBv1Tenant builds a BB v1 (meshstack_buildingblock) resource config with its full dependency chain
// (workspace, project, platform, landing zone, tenant, BBD).
func BBv1Tenant(t *testing.T) (config Config, buildingBlockAddr Traversal) {
	t.Helper()
	workspaceConfig, workspaceAddr := Workspace(t)
	projectConfig, projectAddr := Project(t, workspaceAddr)
	platformConfig, platformAddr, platformTypeAddr := CustomPlatform(t, workspaceAddr)
	landingZoneConfig, landingZoneAddr := LandingZone(t, workspaceAddr, platformAddr, platformTypeAddr)
	tenantConfig, tenantAddr := TenantV4(t, projectAddr, platformAddr, landingZoneAddr)

	var buildingBlockDefinitionAddr Traversal
	buildingBlockDefinitionConfig := Resource{Name: "building_block_v2", Suffix: "_02_tenant"}.TestSupportConfig(t, "").WithFirstBlock(
		ExtractAddress(&buildingBlockDefinitionAddr),
		OwnedByWorkspace(workspaceAddr),
		Descend("spec", "supported_platforms")(SetRawExpr("[{name = %s}]", platformTypeAddr.Join("metadata", "name"))),
	)

	return Resource{Name: "buildingblock"}.Config(t).WithFirstBlock(
		ExtractAddress(&buildingBlockAddr),
		Descend("metadata")(
			Descend("definition_uuid")(SetAddr(buildingBlockDefinitionAddr, "ref", "uuid")),
			Descend("definition_version")(SetAddr(buildingBlockDefinitionAddr, "version_latest", "number")),
			Descend("tenant_identifier")(SetRawExpr(
				`"${%s.metadata.owned_by_workspace}.${%s.metadata.owned_by_project}.${%s.spec.platform_identifier}"`,
				tenantAddr, tenantAddr, tenantAddr)),
		),
		Descend("spec", "inputs", "environment")(SetRawExpr(`{ value_single_select = "dev" }`)),
	).Join(workspaceConfig, projectConfig, platformConfig, landingZoneConfig, tenantConfig, buildingBlockDefinitionConfig), buildingBlockAddr
}
