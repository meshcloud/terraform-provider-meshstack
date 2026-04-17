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
