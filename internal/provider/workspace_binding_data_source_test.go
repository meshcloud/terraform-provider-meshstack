package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/compare"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/knownvalue"
	"github.com/hashicorp/terraform-plugin-testing/statecheck"
	"github.com/hashicorp/terraform-plugin-testing/tfjsonpath"

	"github.com/meshcloud/terraform-provider-meshstack/internal/provider/acctest/testconfig"
)

func TestAccWorkspaceGroupBindingDataSource(t *testing.T) {
	testAccWorkspaceBindingDataSource(t, "workspace_group_binding", "my-user-group")
}

func TestAccWorkspaceUserBindingDataSource(t *testing.T) {
	testAccWorkspaceBindingDataSource(t, "workspace_user_binding", "user@meshcloud.io")
}

func testAccWorkspaceBindingDataSource(t *testing.T, name, subjectName string) {
	t.Helper()
	if !IsMockClientTest() {
		t.Skipf("Skipping: requires subject '%s' in local meshStack", subjectName)
	}

	workspaceConfig, workspaceAddr := testconfig.Workspace(t)

	var bindingAddr testconfig.Traversal
	bindingConfig := testconfig.Resource{Name: name}.Config(t).WithFirstBlock(
		testconfig.ExtractAddress(&bindingAddr),
		testconfig.Descend("target_ref", "name")(testconfig.SetAddr(workspaceAddr, "metadata", "name")),
	)

	var dataSourceAddress testconfig.Traversal
	config := testconfig.DataSource{Name: name}.Config(t).WithFirstBlock(
		testconfig.ExtractAddress(&dataSourceAddress),
		testconfig.Descend("metadata", "name")(testconfig.SetAddr(bindingAddr, "metadata", "name")),
	).Join(bindingConfig, workspaceConfig)

	ApplyAndTest(t, resource.TestCase{
		Steps: []resource.TestStep{
			{
				Config: config.String(),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(dataSourceAddress.String(), tfjsonpath.New("metadata").AtMapKey("name"), knownvalue.StringExact("this-is-an-example")),
					statecheck.ExpectKnownValue(dataSourceAddress.String(), tfjsonpath.New("role_ref").AtMapKey("name"), knownvalue.StringExact("Workspace Member")),
					statecheck.CompareValuePairs(dataSourceAddress.String(), tfjsonpath.New("target_ref").AtMapKey("name"), workspaceAddr.String(), tfjsonpath.New("metadata").AtMapKey("name"), compare.ValuesSame()),
					statecheck.ExpectKnownValue(dataSourceAddress.String(), tfjsonpath.New("subject").AtMapKey("name"), knownvalue.StringExact(subjectName)),
					statecheck.ExpectKnownValue(dataSourceAddress.String(), tfjsonpath.New("expiry_date"), knownvalue.StringExact("2026-12-31")),
				},
			},
		},
	})
}
