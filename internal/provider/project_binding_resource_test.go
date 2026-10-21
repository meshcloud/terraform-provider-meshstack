package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/compare"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/knownvalue"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
	"github.com/hashicorp/terraform-plugin-testing/statecheck"
	"github.com/hashicorp/terraform-plugin-testing/tfjsonpath"

	"github.com/meshcloud/terraform-provider-meshstack/internal/provider/acctest/testconfig"
	"github.com/meshcloud/terraform-provider-meshstack/internal/provider/acctest/xknownvalue"
)

func TestAccProjectUserBinding(t *testing.T) {
	testAccProjectBinding(t, "user", seededUsername, expectUserEnabled)
}

func TestAccProjectGroupBinding(t *testing.T) {
	testAccProjectBinding(t, "group", seededGroupName)
}

func testAccProjectBinding(t *testing.T, subjectKind, subjectName string, subjectChecks ...subjectStateCheck) {
	t.Helper()
	config, bindingAddr, workspaceAddr, bindingName := testconfig.ProjectBinding(t, subjectKind, subjectName)

	ApplyAndTest(t, resource.TestCase{
		Steps: []resource.TestStep{
			{
				Config: config.String(),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction(bindingAddr.String(), plancheck.ResourceActionCreate),
					},
				},
				ConfigStateChecks: append([]statecheck.StateCheck{
					statecheck.ExpectKnownValue(bindingAddr.String(), tfjsonpath.New("metadata").AtMapKey("name"), knownvalue.StringExact(bindingName)),
					statecheck.ExpectKnownValue(bindingAddr.String(), tfjsonpath.New("role_ref").AtMapKey("name"), knownvalue.StringExact("Project Reader")),
					statecheck.ExpectKnownValue(bindingAddr.String(), tfjsonpath.New("target_ref").AtMapKey("name"), xknownvalue.NotEmptyString()),
					statecheck.CompareValuePairs(bindingAddr.String(), tfjsonpath.New("target_ref").AtMapKey("owned_by_workspace"), workspaceAddr.String(), tfjsonpath.New("metadata").AtMapKey("name"), compare.ValuesSame()),
					statecheck.ExpectKnownValue(bindingAddr.String(), tfjsonpath.New("subject").AtMapKey("name"), knownvalue.StringExact(subjectName)),
					statecheck.ExpectKnownValue(bindingAddr.String(), tfjsonpath.New("expiry_date"), knownvalue.StringExact("2099-12-31")),
				}, subjectStateChecks(bindingAddr, subjectChecks)...),
			},
			{
				ResourceName:    bindingAddr.String(),
				ImportState:     true,
				ImportStateId:   bindingName,
				ImportStateKind: resource.ImportBlockWithID,
			},
		},
	})
}
