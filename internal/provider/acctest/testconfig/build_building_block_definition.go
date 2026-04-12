package testconfig

import (
	"fmt"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
)

func BuildBBDTerraformConfig(t *testing.T) (config Config, bbdAddr Traversal) {
	t.Helper()
	workspaceConfig, workspaceAddr := BuildWorkspaceConfig(t)
	exampleResource := Resource{Name: "building_block_definition", Suffix: "_01_terraform"}

	var environmentTagAddr, costCenterTagAddr, dependencyBBDAddr Traversal

	tagSuffix := acctest.RandString(8)
	envTagConfig := exampleResource.TestSupportConfig(t, "_tag-environment").WithFirstBlock(t,
		ExtractIdentifier(&environmentTagAddr),
		Traverse(t, "spec", "key")(SetString("environment-"+tagSuffix)),
	)
	costTagConfig := exampleResource.TestSupportConfig(t, "_tag-cost-center").WithFirstBlock(t,
		ExtractIdentifier(&costCenterTagAddr),
		Traverse(t, "spec", "key")(SetString("cost-center-"+tagSuffix)),
	)

	depBBDConfig := exampleResource.TestSupportConfig(t, "_dependency-bbd").WithFirstBlock(t,
		ExtractIdentifier(&dependencyBBDAddr),
		OwnedByWorkspace(t, workspaceAddr),
	)

	return exampleResource.Config(t).WithFirstBlock(t,
			ExtractIdentifier(&bbdAddr),
			OwnedByWorkspace(t, workspaceAddr),
			Traverse(t, "metadata", "tags")(
				SetRawExpr(fmt.Sprintf(`{(%s) = ["dev", "prod"], (%s) = ["cc-123"]}`,
					environmentTagAddr.Join("spec", "key"),
					costCenterTagAddr.Join("spec", "key"),
				)),
			),
			Traverse(t, "spec")(
				Traverse(t, "notification_subscribers")(SetRawExpr(`["email:ops@example.com"]`)),
				Traverse(t, "symbol")(SetRawExpr(`provider::meshstack::load_image_file("testdata/images/image.png")`)),
			),
			Traverse(t, "version_spec")(
				Traverse(t, "inputs", "some-file.yaml", "argument")(SetRawExpr(`jsonencode(provider::meshstack::encode_file("some-content"))`)),
				Traverse(t, "dependency_refs")(SetRawExpr(dependencyBBDAddr.Format("[%s.ref]"))),
			),
		).Join(workspaceConfig, envTagConfig, costTagConfig, depBBDConfig),
		bbdAddr
}

func implAttrFromSuffix(suffix string) string {
	_, after, _ := strings.Cut(suffix, "_")
	return after
}

func BuildBBDWithIntegrationConfig(t *testing.T, suffix string) (config Config, bbdAddr Traversal) {
	t.Helper()
	workspaceConfig, workspaceAddr := BuildWorkspaceConfig(t)
	exampleResource := Resource{Name: "building_block_definition", Suffix: "_" + suffix}

	var integrationAddr Traversal
	integrationConfig := exampleResource.TestSupportConfig(t, "_integration").WithFirstBlock(t,
		ExtractIdentifier(&integrationAddr),
		OwnedByWorkspace(t, workspaceAddr),
	)

	return exampleResource.Config(t).WithFirstBlock(t,
		ExtractIdentifier(&bbdAddr),
		OwnedByWorkspace(t, workspaceAddr),
		Traverse(t, "version_spec", "implementation", implAttrFromSuffix(suffix), "integration_ref")(
			SetRawExpr(integrationAddr.Join("ref").String()),
		),
	).Join(workspaceConfig, integrationConfig), bbdAddr
}

func BuildBBDManualConfig(t *testing.T) (config Config, bbdAddr Traversal) {
	t.Helper()
	workspaceConfig, workspaceAddr := BuildWorkspaceConfig(t)
	return Resource{Name: "building_block_definition", Suffix: "_03_manual"}.Config(t).WithFirstBlock(t,
		ExtractIdentifier(&bbdAddr),
		OwnedByWorkspace(t, workspaceAddr),
	).Join(workspaceConfig), bbdAddr
}

func BuildBBDGitlabPipelineConfig(t *testing.T) (config Config, bbdAddr Traversal) {
	t.Helper()
	displayNameSuffix := acctest.RandString(8)
	config, bbdAddr = BuildBBDWithIntegrationConfig(t, "05_gitlab_pipeline")
	return config.WithFirstBlock(t,
		Traverse(t, "spec", "display_name")(SetString("Example Building Block "+displayNameSuffix)),
	), bbdAddr
}
