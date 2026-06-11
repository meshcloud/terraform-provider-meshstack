package testconfig

import (
	"testing"
)

// localBuildingBlockRunnerUuid is the runner UUID registered on a locally running meshStack.
const localBuildingBlockRunnerUuid = "46b7c17a-61f0-4062-9601-5785e60ce11f"

// BBDRunnerRef returns an ExpressionConsumer that sets version_spec.runner_ref to the given UUID.
// Use this to override the default runner when the target meshStack uses a non-standard runner.
func BBDRunnerRef(uuid string) ExpressionConsumer {
	return Descend("version_spec", "runner_ref")(
		SetRawExpr(`{kind = "meshBuildingBlockRunner", uuid = %q}`, uuid),
	)
}

// BBv2Workspace builds a workspace-level BB v2 resource config with its BBD and workspace.
func BBv2Workspace(t *testing.T) (config Config, buildingBlockAddr Traversal) {
	t.Helper()
	return bbv2Workspace(t)
}

// BBv2WorkspaceLocal is like BBv2Workspace but sets the BBD runner_ref to the local meshStack runner.
func BBv2WorkspaceLocal(t *testing.T) (config Config, buildingBlockAddr Traversal) {
	t.Helper()
	return bbv2Workspace(t, BBDRunnerRef(localBuildingBlockRunnerUuid))
}

func bbv2Workspace(t *testing.T, extraBBDMods ...ExpressionConsumer) (config Config, buildingBlockAddr Traversal) {
	t.Helper()
	workspaceConfig, workspaceAddr := Workspace(t)
	var buildingBlockDefinitionAddr Traversal
	buildingBlockDefinitionConfig := Resource{Name: "building_block_v2", Suffix: "_01_workspace"}.TestSupportConfig(t, "").WithFirstBlock(
		append([]ExpressionConsumer{
			ExtractAddress(&buildingBlockDefinitionAddr),
			OwnedByWorkspace(workspaceAddr),
		}, extraBBDMods...)...,
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
	return bbv2Tenant(t)
}

// BBv2TenantLocal is like BBv2Tenant but sets the BBD runner_ref to the local meshStack runner.
func BBv2TenantLocal(t *testing.T) (config Config, buildingBlockAddr Traversal) {
	t.Helper()
	return bbv2Tenant(t, BBDRunnerRef(localBuildingBlockRunnerUuid))
}

func bbv2Tenant(t *testing.T, extraBBDMods ...ExpressionConsumer) (config Config, buildingBlockAddr Traversal) {
	t.Helper()
	workspaceConfig, workspaceAddr := Workspace(t)
	projectConfig, projectAddr := Project(t, workspaceAddr)
	platformConfig, platformAddr, platformTypeAddr := CustomPlatform(t, workspaceAddr)
	landingZoneConfig, landingZoneAddr := LandingZone(t, workspaceAddr, platformAddr, platformTypeAddr)

	tenantConfig, tenantAddr := TenantV4(t, projectAddr, platformAddr, landingZoneAddr)

	var buildingBlockDefinitionAddr Traversal
	buildingBlockDefinitionConfig := Resource{Name: "building_block_v2", Suffix: "_02_tenant"}.TestSupportConfig(t, "").WithFirstBlock(
		append([]ExpressionConsumer{
			ExtractAddress(&buildingBlockDefinitionAddr),
			OwnedByWorkspace(workspaceAddr),
			Descend("spec", "supported_platforms")(SetRawExpr("[{name = %s}]", platformTypeAddr.Join("metadata", "name"))),
		}, extraBBDMods...)...,
	)

	return Resource{Name: "building_block_v2", Suffix: "_02_tenant"}.Config(t).WithFirstBlock(
		ExtractAddress(&buildingBlockAddr),
		Descend("spec", "building_block_definition_version_ref")(SetAddr(buildingBlockDefinitionAddr, "version_latest")),
		Descend("spec", "target_ref")(SetAddr(tenantAddr, "ref")),
	).Join(workspaceConfig, projectConfig, platformConfig, landingZoneConfig, tenantConfig, buildingBlockDefinitionConfig), buildingBlockAddr
}
