package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/knownvalue"
	"github.com/hashicorp/terraform-plugin-testing/statecheck"
	"github.com/hashicorp/terraform-plugin-testing/tfjsonpath"

	"github.com/meshcloud/terraform-provider-meshstack/internal/provider/acctest/testconfig"
	"github.com/meshcloud/terraform-provider-meshstack/internal/provider/acctest/xknownvalue"
)

func TestAccProjectUserBinding(t *testing.T) {
	projectConfig, projectAddr, workspaceAddr := testconfig.ProjectAndWorkspace(t)

	bindingName := "test-pub-" + acctest.RandString(8)
	workspaceBindingConfig, workspaceBindingAddr := testconfig.WorkspaceUserBinding(t, workspaceAddr, SeededUsername)

	var resourceAddress testconfig.Traversal
	config := testconfig.Resource{Name: "project_user_binding"}.Config(t).WithFirstBlock(
		testconfig.ExtractAddress(&resourceAddress),
		testconfig.Descend("metadata", "name")(testconfig.SetString(bindingName)),
		testconfig.Descend("subject", "name")(testconfig.SetString(SeededUsername)),
		testconfig.Descend("depends_on")(testconfig.SetRawExpr("[%s]", workspaceBindingAddr)),
		testconfig.Descend("target_ref")(
			testconfig.Descend("owned_by_workspace")(testconfig.SetAddr(workspaceAddr, "metadata", "name")),
			testconfig.Descend("name")(testconfig.SetAddr(projectAddr, "metadata", "name")),
		),
	).Join(projectConfig, workspaceBindingConfig)

	ApplyAndTest(t, resource.TestCase{
		Steps: seededUserBindingCreateImportSteps(config, resourceAddress, bindingName,
			statecheck.ExpectKnownValue(resourceAddress.String(), tfjsonpath.New("role_ref").AtMapKey("name"), knownvalue.StringExact("Project Reader")),
			statecheck.ExpectKnownValue(resourceAddress.String(), tfjsonpath.New("target_ref").AtMapKey("name"), xknownvalue.NotEmptyString()),
		),
	})
}
