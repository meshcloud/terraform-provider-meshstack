package testconfig

import (
	"testing"
)

// BBv2Workspace builds a workspace-level BB v2 resource config with its BBD and workspace.
func BBv2Workspace(t *testing.T) (config Config, buildingBlockAddr Traversal) {
	t.Helper()
	workspaceConfig, workspaceAddr := Workspace(t)
	var buildingBlockDefinitionAddr Traversal
	buildingBlockDefinitionConfig := Resource{Name: "building_block_v2", Suffix: "_01_workspace"}.TestSupportConfig(t, "").WithFirstBlock(
		ExtractAddress(&buildingBlockDefinitionAddr),
		OwnedByWorkspace(workspaceAddr),
	)
	return Resource{Name: "building_block_v2", Suffix: "_01_workspace"}.Config(t).WithFirstBlock(
		ExtractAddress(&buildingBlockAddr),
		Descend("spec", "building_block_definition_version_ref")(SetAddr(buildingBlockDefinitionAddr, "version_latest")),
		Descend("spec", "target_ref")(SetAddr(workspaceAddr, "ref")),
	).Join(workspaceConfig, buildingBlockDefinitionConfig), buildingBlockAddr
}

// BBv2Tenant builds a tenant-level BB v2 resource config with its full dependency chain
// (workspace, project, platform, landing zone, tenant, BBD).
func BBv2Tenant(t *testing.T) (config Config, buildingBlockAddr Traversal) {
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

	return Resource{Name: "building_block_v2", Suffix: "_02_tenant"}.Config(t).WithFirstBlock(
		ExtractAddress(&buildingBlockAddr),
		Descend("spec", "building_block_definition_version_ref")(SetAddr(buildingBlockDefinitionAddr, "version_latest")),
		Descend("spec", "target_ref")(SetAddr(tenantAddr, "ref")),
	).Join(workspaceConfig, projectConfig, platformConfig, landingZoneConfig, tenantConfig, buildingBlockDefinitionConfig), buildingBlockAddr
}
