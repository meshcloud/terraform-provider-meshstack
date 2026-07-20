package testconfig

import (
	"testing"
)

// BBWorkspace builds a workspace, a building block definition owned by it, and a v3 building
// block on that workspace wired to the definition's latest version. workspaceAddr is the
// underlying Workspace(t) address, returned so callers can attach further workspace-scoped
// resources without rebuilding the workspace.
func BBWorkspace(t *testing.T) (config Config, buildingBlockAddr Traversal, buildingBlockDefinitionAddr Traversal, workspaceAddr Traversal) {
	t.Helper()
	workspaceConfig, workspaceAddr := Workspace(t)
	buildingBlockDefinitionConfig := Resource{Name: "building_block", Suffix: "_01_workspace"}.TestSupportConfig(t, "").WithFirstBlock(
		ExtractAddress(&buildingBlockDefinitionAddr),
		OwnedByWorkspace(workspaceAddr),
	)
	return Resource{Name: "building_block", Suffix: "_01_workspace"}.Config(t).WithFirstBlock(
		ExtractAddress(&buildingBlockAddr),
		// Wire only the version uuid (not the whole version_latest object, which carries
		// content_hash). content_hash is a TF-only field that can't be recovered on import,
		// so leaving it unset keeps ImportBlockWithID a no-op. Tests that exercise content_hash
		// set it explicitly in later steps.
		Descend("spec", "building_block_definition_version_ref")(SetRawExpr(`{ uuid = %s }`, buildingBlockDefinitionAddr.Join("version_latest", "uuid"))),
		Descend("spec", "target_ref")(SetAddr(workspaceAddr, "ref")),
	).Join(workspaceConfig, buildingBlockDefinitionConfig), buildingBlockAddr, buildingBlockDefinitionAddr, workspaceAddr
}

// BBTenant builds a workspace (+project/platform/landing-zone/tenant) and a v3 building block
// targeting that tenant. The building block definition uses the terraform implementation; callers
// pass terraformRepoUrl (a loopback git smart-HTTP URL to the committed bare repo, served by the
// test's git-http-backend — see git_http_server_test.go) so the real tf-block-runner can clone and
// run OpenTofu offline in acceptance mode. In mock mode the URL is unused. workspaceAddr
// is the underlying Workspace(t) address, returned so callers can attach further workspace-scoped
// resources without rebuilding the workspace.
func BBTenant(t *testing.T, terraformRepoUrl string) (config Config, buildingBlockAddr Traversal, workspaceAddr Traversal) {
	t.Helper()
	workspaceConfig, workspaceAddr := Workspace(t)
	projectConfig, projectAddr := Project(t, workspaceAddr)
	platformConfig, platformAddr, platformTypeAddr := CustomPlatform(t, workspaceAddr)
	landingZoneConfig, landingZoneAddr := LandingZone(t, workspaceAddr, platformAddr, platformTypeAddr)

	var tenantAddr Traversal
	tenantConfig := Resource{Name: "tenant"}.Config(t).FirstBlockOnly().WithFirstBlock(
		ExtractAddress(&tenantAddr),
		Descend("metadata")(
			Descend("owned_by_workspace")(SetAddr(projectAddr, "metadata", "owned_by_workspace")),
			Descend("owned_by_project")(SetAddr(projectAddr, "metadata", "name")),
		),
		Descend("spec")(
			Descend("platform_ref")(SetRawExpr("%s", platformAddr.Join("ref"))),
			Descend("landing_zone_ref")(SetAddr(landingZoneAddr, "ref")),
		),
	)

	var buildingBlockDefinitionAddr Traversal
	buildingBlockDefinitionConfig := Resource{Name: "building_block", Suffix: "_02_tenant"}.TestSupportConfig(t, "").WithFirstBlock(
		ExtractAddress(&buildingBlockDefinitionAddr),
		OwnedByWorkspace(workspaceAddr),
		Descend("spec", "supported_platforms")(SetRawExpr("[{name = %s}]", platformTypeAddr.Join("metadata", "name"))),
		// Point the terraform implementation at the committed bare repo served over loopback git
		// smart-HTTP so the real tf-block-runner clones and runs OpenTofu offline. The static example
		// URL in the .tf is a docs placeholder; in mock mode this value is unused.
		Descend("version_spec", "implementation", "terraform", "repository_url")(SetRawExpr("%q", terraformRepoUrl)),
	)

	return Resource{Name: "building_block", Suffix: "_02_tenant"}.Config(t).WithFirstBlock(
		ExtractAddress(&buildingBlockAddr),
		// Wire only the version uuid (see BBWorkspace) so content_hash stays unset and
		// ImportBlockWithID remains a no-op.
		Descend("spec", "building_block_definition_version_ref")(SetRawExpr(`{ uuid = %s }`, buildingBlockDefinitionAddr.Join("version_latest", "uuid"))),
		Descend("spec", "target_ref")(SetAddr(tenantAddr, "ref")),
	).Join(workspaceConfig, projectConfig, platformConfig, landingZoneConfig, tenantConfig, buildingBlockDefinitionConfig), buildingBlockAddr, workspaceAddr
}
