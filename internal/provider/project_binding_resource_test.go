package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/knownvalue"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
	"github.com/hashicorp/terraform-plugin-testing/statecheck"
	"github.com/hashicorp/terraform-plugin-testing/tfjsonpath"
)

func TestAccProjectUserBinding(t *testing.T) {
	testAccProjectBinding(t, "user", seededUsername, expectUserEnabled, expectApprovalAssigned)
}

func TestAccProjectGroupBinding(t *testing.T) {
	testAccProjectBinding(t, "group", seededGroupName, expectApprovalAssigned)
}

func testAccProjectBinding(t *testing.T, subjectKind, subjectName string, subjectChecks ...subjectStateCheck) {
	t.Helper()
	suffix := acctest.RandString(8)
	vars := hclStepVars{"suffix": suffix, "subject_kind": subjectKind, "subject_name": subjectName}
	bindingAddress := "meshstack_project_" + subjectKind + "_binding.example"

	ApplyAndTest(t, resource.TestCase{
		Steps: []resource.TestStep{
			{
				Config: hclStep(t, "resources/project_binding", 1, vars),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction(bindingAddress, plancheck.ResourceActionCreate),
					},
				},
				ConfigStateChecks: append([]statecheck.StateCheck{
					statecheck.ExpectKnownValue(bindingAddress, tfjsonpath.New("metadata").AtMapKey("name"), knownvalue.StringExact("example-project-"+subjectKind+"-binding")),
					statecheck.ExpectKnownValue(bindingAddress, tfjsonpath.New("role_ref").AtMapKey("name"), knownvalue.StringExact("Project Reader")),
					statecheck.ExpectKnownValue(bindingAddress, tfjsonpath.New("target_ref").AtMapKey("name"), knownvalue.StringExact("test-proj-"+suffix)),
					statecheck.ExpectKnownValue(bindingAddress, tfjsonpath.New("target_ref").AtMapKey("owned_by_workspace"), knownvalue.StringExact("test-ws-"+suffix)),
					statecheck.ExpectKnownValue(bindingAddress, tfjsonpath.New("subject").AtMapKey("name"), knownvalue.StringExact(subjectName)),
					statecheck.ExpectKnownValue(bindingAddress, tfjsonpath.New("expiry_date"), knownvalue.StringExact("2099-12-31")),
				}, subjectStateChecks(bindingAddress, subjectChecks)...),
			},
			{
				ResourceName:    bindingAddress,
				ImportState:     true,
				ImportStateId:   "example-project-" + subjectKind + "-binding",
				ImportStateKind: resource.ImportBlockWithID,
			},
		},
	})
}
