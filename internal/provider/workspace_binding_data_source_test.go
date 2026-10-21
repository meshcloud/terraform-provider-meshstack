package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/compare"
	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/knownvalue"
	"github.com/hashicorp/terraform-plugin-testing/statecheck"
	"github.com/hashicorp/terraform-plugin-testing/tfjsonpath"

	"github.com/meshcloud/terraform-provider-meshstack/internal/provider/acctest/testconfig"
)

func TestAccWorkspaceGroupBindingDataSource(t *testing.T) {
	testAccWorkspaceBindingDataSource(t, "workspace_group_binding", seededGroupName)
}

func TestAccWorkspaceUserBindingDataSource(t *testing.T) {
	testAccWorkspaceBindingDataSource(t, "workspace_user_binding", seededUsername, expectUserEnabled)
}

func testAccWorkspaceBindingDataSource(t *testing.T, name, subjectName string, subjectChecks ...subjectStateCheck) {
	t.Helper()
	workspaceConfig, workspaceAddr := testconfig.Workspace(t)

	bindingName := "test-binding-" + acctest.RandString(8)
	var bindingAddr testconfig.Traversal
	bindingConfig := testconfig.Resource{Name: name}.Config(t).WithFirstBlock(
		testconfig.ExtractAddress(&bindingAddr),
		testconfig.Descend("metadata", "name")(testconfig.SetString(bindingName)),
		testconfig.Descend("target_ref", "name")(testconfig.SetAddr(workspaceAddr, "metadata", "name")),
		testconfig.Descend("subject", "name")(testconfig.SetString(subjectName)),
	).Join(workspaceConfig)

	testAccBindingDataSource(t, name, bindingConfig, bindingAddr, "Workspace Member", "Workspace Manager", subjectChecks...)
}

// testAccBindingDataSource reads the binding by name, then by target_ref, subject and role, and
// expects no match for otherRole.
func testAccBindingDataSource(t *testing.T, name string, bindingConfig testconfig.Config, bindingAddr testconfig.Traversal, role, otherRole string, subjectChecks ...subjectStateCheck) {
	t.Helper()
	var dataSourceAddress testconfig.Traversal
	example := testconfig.DataSource{Name: name}.Config(t).WithFirstBlock(testconfig.ExtractAddress(&dataSourceAddress))
	byName := example.WithFirstBlock(
		testconfig.Descend("metadata", "name")(testconfig.SetAddr(bindingAddr, "metadata", "name")),
	).Join(bindingConfig)
	byTargetSubjectAndRole := func(role string) testconfig.Config {
		return example.WithFirstBlock(
			testconfig.Descend("metadata")(testconfig.RemoveKey()),
			testconfig.Descend("target_ref")(testconfig.SetAddr(bindingAddr, "target_ref")),
			testconfig.Descend("subject")(testconfig.SetAddr(bindingAddr, "subject")),
			testconfig.Descend("role_ref")(testconfig.SetRawExpr("{ name = %q }", role)),
		).Join(bindingConfig)
	}

	expectFound := []statecheck.StateCheck{
		statecheck.ExpectKnownValue(dataSourceAddress.String(), tfjsonpath.New("role_ref").AtMapKey("name"), knownvalue.StringExact(role)),
		statecheck.ExpectKnownValue(dataSourceAddress.String(), tfjsonpath.New("expiry_date"), knownvalue.StringExact("2099-12-31")),
		statecheck.ExpectKnownValue(dataSourceAddress.String(), tfjsonpath.New("expired"), knownvalue.Bool(false)),
	}
	expectFound = append(expectFound, subjectStateChecks(dataSourceAddress, subjectChecks)...)
	for _, attribute := range []string{"metadata", "target_ref", "subject"} {
		expectFound = append(expectFound, statecheck.CompareValuePairs(
			dataSourceAddress.String(), tfjsonpath.New(attribute), bindingAddr.String(), tfjsonpath.New(attribute), compare.ValuesSame()))
	}
	ApplyAndTest(t, resource.TestCase{
		Steps: []resource.TestStep{
			{Config: byName.String(), ConfigStateChecks: expectFound},
			{Config: byTargetSubjectAndRole(role).String(), ConfigStateChecks: expectFound},
			{
				Config:      byTargetSubjectAndRole(otherRole).String(),
				ExpectError: expectErrorWords("binding", "matches"),
			},
		},
	})
}
