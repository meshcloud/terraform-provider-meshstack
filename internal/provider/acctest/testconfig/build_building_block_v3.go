package testconfig

import (
	"fmt"
	"testing"
)

func BuildBBv3WorkspaceConfig(t *testing.T) (config Config, bbAddr Traversal) {
	t.Helper()
	workspaceConfig, workspaceAddr := BuildWorkspaceConfig(t)
	var bbdAddr Traversal
	bbdConfig := Resource{Name: "building_block_v3", Suffix: "_01_workspace"}.TestSupportConfig(t, "").WithFirstBlock(t,
		ExtractIdentifier(&bbdAddr),
		OwnedByWorkspace(t, workspaceAddr),
	)
	return Resource{Name: "building_block_v3", Suffix: "_01_workspace"}.Config(t).WithFirstBlock(t,
		ExtractIdentifier(&bbAddr),
		Traverse(t, "spec", "building_block_definition_version_ref")(SetRawExpr(bbdAddr.Join("version_latest").String())),
		Traverse(t, "spec", "target_ref")(SetRawExpr(workspaceAddr.Join("ref").String())),
	).Join(workspaceConfig, bbdConfig), bbAddr
}

func BuildBBv3TenantConfig(t *testing.T) (config Config, bbAddr Traversal) {
	t.Helper()
	workspaceConfig, workspaceAddr := BuildWorkspaceConfig(t)
	projectConfig, projectAddr := BuildProjectConfig(t, workspaceAddr)
	platformConfig, platformAddr, platformTypeAddr := BuildCustomPlatformConfig(t, workspaceAddr)
	landingZoneConfig, landingZoneAddr := BuildLandingZoneConfig(t, workspaceAddr, platformAddr, platformTypeAddr)

	var tenantAddr Traversal
	tenantConfig := Resource{Name: "tenant_v4"}.Config(t).WithFirstBlock(t,
		ExtractIdentifier(&tenantAddr),
		Traverse(t, "metadata")(
			Traverse(t, "owned_by_workspace")(SetRawExpr(projectAddr.Join("metadata", "owned_by_workspace").String())),
			Traverse(t, "owned_by_project")(SetRawExpr(projectAddr.Join("metadata", "name").String())),
		),
		Traverse(t, "spec")(
			Traverse(t, "platform_identifier")(SetRawExpr(platformAddr.Join("identifier").String())),
			Traverse(t, "landing_zone_identifier")(SetRawExpr(landingZoneAddr.Join("metadata", "name").String())),
		),
	)

	var bbdAddr Traversal
	bbdConfig := Resource{Name: "building_block_v3", Suffix: "_02_tenant"}.TestSupportConfig(t, "").WithFirstBlock(t,
		ExtractIdentifier(&bbdAddr),
		OwnedByWorkspace(t, workspaceAddr),
		Traverse(t, "spec", "supported_platforms")(SetRawExpr(fmt.Sprintf("[{name = %s}]", platformTypeAddr.Join("metadata", "name")))),
	)

	return Resource{Name: "building_block_v3", Suffix: "_02_tenant"}.Config(t).WithFirstBlock(t,
		ExtractIdentifier(&bbAddr),
		Traverse(t, "spec", "building_block_definition_version_ref")(SetRawExpr(bbdAddr.Join("version_latest").String())),
		Traverse(t, "spec", "target_ref")(SetRawExpr(tenantAddr.Join("ref").String())),
	).Join(workspaceConfig, projectConfig, platformConfig, landingZoneConfig, tenantConfig, bbdConfig), bbAddr
}
