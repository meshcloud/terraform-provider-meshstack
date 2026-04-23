package testconfig

import (
	"testing"
)

func BBv3Workspace(t *testing.T) (config Config, buildingBlockAddr Traversal) {
	t.Helper()
	workspaceConfig, workspaceAddr := Workspace(t)
	var buildingBlockDefinitionAddr Traversal
	buildingBlockDefinitionConfig := Resource{Name: "building_block_v3", Suffix: "_01_workspace"}.TestSupportConfig(t, "").WithFirstBlock(
		ExtractAddress(&buildingBlockDefinitionAddr),
		OwnedByWorkspace(workspaceAddr),
	)
	return Resource{Name: "building_block_v3", Suffix: "_01_workspace"}.Config(t).WithFirstBlock(
		ExtractAddress(&buildingBlockAddr),
		Descend("spec", "building_block_definition_version_ref")(SetAddr(buildingBlockDefinitionAddr, "version_latest")),
		Descend("spec", "target_ref")(SetAddr(workspaceAddr, "ref")),
	).Join(workspaceConfig, buildingBlockDefinitionConfig), buildingBlockAddr
}

func BBv3Tenant(t *testing.T) (config Config, buildingBlockAddr Traversal) {
	t.Helper()
	workspaceConfig, workspaceAddr := Workspace(t)
	projectConfig, projectAddr := Project(t, workspaceAddr)
	platformConfig, platformAddr, platformTypeAddr := CustomPlatform(t, workspaceAddr)
	landingZoneConfig, landingZoneAddr := LandingZone(t, workspaceAddr, platformAddr, platformTypeAddr)

	var tenantAddr Traversal
	tenantConfig := Resource{Name: "tenant_v4"}.Config(t).WithFirstBlock(
		ExtractAddress(&tenantAddr),
		Descend("metadata")(
			Descend("owned_by_workspace")(SetAddr(projectAddr, "metadata", "owned_by_workspace")),
			Descend("owned_by_project")(SetAddr(projectAddr, "metadata", "name")),
		),
		Descend("spec")(
			Descend("platform_identifier")(SetAddr(platformAddr, "identifier")),
			Descend("landing_zone_identifier")(SetAddr(landingZoneAddr, "metadata", "name")),
		),
	)

	var buildingBlockDefinitionAddr Traversal
	buildingBlockDefinitionConfig := Resource{Name: "building_block_v3", Suffix: "_02_tenant"}.TestSupportConfig(t, "").WithFirstBlock(
		ExtractAddress(&buildingBlockDefinitionAddr),
		OwnedByWorkspace(workspaceAddr),
		Descend("spec", "supported_platforms")(SetRawExpr("[{name = %s}]", platformTypeAddr.Join("metadata", "name"))),
	)

	return Resource{Name: "building_block_v3", Suffix: "_02_tenant"}.Config(t).WithFirstBlock(
		ExtractAddress(&buildingBlockAddr),
		Descend("spec", "building_block_definition_version_ref")(SetAddr(buildingBlockDefinitionAddr, "version_latest")),
		Descend("spec", "target_ref")(SetAddr(tenantAddr, "ref")),
	).Join(workspaceConfig, projectConfig, platformConfig, landingZoneConfig, tenantConfig, buildingBlockDefinitionConfig), buildingBlockAddr
}
