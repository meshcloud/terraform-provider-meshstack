package testconfig

import (
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
)

// BBDTerraform builds a complete BBD config with Terraform implementation type,
// including required tag definitions and a dependency BBD, owned by a new test workspace.
func BBDTerraform(t *testing.T) (config Config, buildingBlockDefinitionAddr Traversal) {
	t.Helper()
	workspaceConfig, workspaceAddr := Workspace(t)
	exampleResource := Resource{Name: "building_block_definition", Suffix: "_01_terraform"}

	var environmentTagAddr, costCenterTagAddr, dependencyBBDAddr Traversal

	tagSuffix := acctest.RandString(8)
	envTagConfig := exampleResource.TestSupportConfig(t, "_tag-environment").WithFirstBlock(
		ExtractAddress(&environmentTagAddr),
		Descend("spec", "key")(SetString("environment-"+tagSuffix)),
	)
	costTagConfig := exampleResource.TestSupportConfig(t, "_tag-cost-center").WithFirstBlock(
		ExtractAddress(&costCenterTagAddr),
		Descend("spec", "key")(SetString("cost-center-"+tagSuffix)),
	)

	depBBDConfig := exampleResource.TestSupportConfig(t, "_dependency-bbd").WithFirstBlock(
		ExtractAddress(&dependencyBBDAddr),
		OwnedByWorkspace(workspaceAddr),
	)

	return exampleResource.Config(t).WithFirstBlock(
			ExtractAddress(&buildingBlockDefinitionAddr),
			OwnedByWorkspace(workspaceAddr),
			Descend("metadata", "tags")(
				SetRawExpr(`{(%s) = ["dev", "prod"], (%s) = ["cc-123"]}`,
					environmentTagAddr.Join("spec", "key"),
					costCenterTagAddr.Join("spec", "key"),
				),
			),
			Descend("spec")(
				Descend("notification_subscribers")(SetRawExpr(`["email:ops@example.com"]`)),
				Descend("symbol")(SetRawExpr(`provider::meshstack::load_image_file("testdata/images/image.png")`)),
			),
			Descend("version_spec")(
				Descend("inputs", "some-file.yaml", "argument")(SetRawExpr(`jsonencode(provider::meshstack::encode_file("some-content"))`)),
				Descend("dependency_refs")(SetRawExpr("[%s.ref]", dependencyBBDAddr)),
			),
		).Join(workspaceConfig, envTagConfig, costTagConfig, depBBDConfig),
		buildingBlockDefinitionAddr
}

// BBDWithIntegration builds a BBD config with the given implementation type suffix
// (e.g. "02_github_workflows"), wired to a test integration and owned by a new test workspace.
func BBDWithIntegration(t *testing.T, suffix string) (config Config, buildingBlockDefinitionAddr Traversal) {
	t.Helper()
	workspaceConfig, workspaceAddr := Workspace(t)
	exampleResource := Resource{Name: "building_block_definition", Suffix: "_" + suffix}

	var integrationAddr Traversal
	integrationConfig := exampleResource.TestSupportConfig(t, "_integration").WithFirstBlock(
		ExtractAddress(&integrationAddr),
		OwnedByWorkspace(workspaceAddr),
	)

	_, implementationAttr, _ := strings.Cut(suffix, "_")

	return exampleResource.Config(t).WithFirstBlock(
		ExtractAddress(&buildingBlockDefinitionAddr),
		OwnedByWorkspace(workspaceAddr),
		Descend("version_spec", "implementation", implementationAttr, "integration_ref")(
			SetAddr(integrationAddr, "ref"),
		),
	).Join(workspaceConfig, integrationConfig), buildingBlockDefinitionAddr
}

// BBDGithubTwoIntegrations builds a github_workflows BBD wired to a first test integration ("A"),
// plus a second github integration ("B") owned by the same workspace. It returns the config, the
// BBD address, and integration B's address so a test can switch the BBD's integration_ref from A to
// B (e.g. to assert that re-drafting a released version with a new integration keeps the released
// version immutable). Both integrations reuse the github integration test-support file; the second
// is renamed and given a distinct display name so the backend treats it as a different integration.
func BBDGithubTwoIntegrations(t *testing.T) (config Config, buildingBlockDefinitionAddr Traversal, integrationBAddr Traversal) {
	t.Helper()
	workspaceConfig, workspaceAddr := Workspace(t)
	exampleResource := Resource{Name: "building_block_definition", Suffix: "_02_github_workflows"}

	var integrationAAddr Traversal
	integrationAConfig := exampleResource.TestSupportConfig(t, "_integration").WithFirstBlock(
		ExtractAddress(&integrationAAddr),
		OwnedByWorkspace(workspaceAddr),
	)
	integrationBConfig := exampleResource.TestSupportConfig(t, "_integration").WithFirstBlock(
		RenameKey("github_b"),
		ExtractAddress(&integrationBAddr),
		OwnedByWorkspace(workspaceAddr),
		Descend("spec", "display_name")(SetString("GitHub Integration B")),
	)

	return exampleResource.Config(t).WithFirstBlock(
		ExtractAddress(&buildingBlockDefinitionAddr),
		OwnedByWorkspace(workspaceAddr),
		Descend("version_spec", "implementation", "github_workflows", "integration_ref")(
			SetAddr(integrationAAddr, "ref"),
		),
		// Depend on integration A explicitly: once the BBD switches its integration_ref to B, the
		// released version still pins A on the backend (which refuses integration deletion while
		// referenced). This keeps A scheduled for destruction after the BBD so teardown succeeds; B is
		// already implicitly ordered via the integration_ref reference.
		Descend("depends_on")(SetRawExpr("[%s]", integrationAAddr)),
	).Join(workspaceConfig, integrationAConfig, integrationBConfig), buildingBlockDefinitionAddr, integrationBAddr
}

// BBDManual builds a BBD config with manual implementation type, owned by a new test workspace.
func BBDManual(t *testing.T) (config Config, buildingBlockDefinitionAddr Traversal) {
	t.Helper()
	workspaceConfig, workspaceAddr := Workspace(t)
	return Resource{Name: "building_block_definition", Suffix: "_03_manual"}.Config(t).WithFirstBlock(
		ExtractAddress(&buildingBlockDefinitionAddr),
		OwnedByWorkspace(workspaceAddr),
	).Join(workspaceConfig), buildingBlockDefinitionAddr
}

// BBDGitlabPipeline builds a BBD config with GitLab pipeline implementation type.
func BBDGitlabPipeline(t *testing.T) (config Config, buildingBlockDefinitionAddr Traversal) {
	t.Helper()
	displayNameSuffix := acctest.RandString(8)
	config, buildingBlockDefinitionAddr = BBDWithIntegration(t, "05_gitlab_pipeline")
	return config.WithFirstBlock(
		Descend("spec", "display_name")(SetString("Example Building Block " + displayNameSuffix)),
	), buildingBlockDefinitionAddr
}
