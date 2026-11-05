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

func TestAccWorkspaceUserBinding(t *testing.T) {
	t.Parallel()
	testAccWorkspaceBinding(t, "user", seededUsername, expectUserEnabled)
}

func TestAccWorkspaceGroupBinding(t *testing.T) {
	t.Parallel()
	testAccWorkspaceBinding(t, "group", seededGroupName)
}

func testAccWorkspaceBinding(t *testing.T, subjectKind, subjectName string, subjectChecks ...subjectStateCheck) {
	t.Helper()

	t.Run("with_expiry_date", func(t *testing.T) {
		suffix := acctest.RandString(8)
		vars := hclStepVars{"suffix": suffix, "subject_kind": subjectKind, "subject_name": subjectName}
		bindingName := "test-wb-" + suffix
		resourceAddress := "meshstack_workspace_" + subjectKind + "_binding.example"

		ApplyAndTest(t, resource.TestCase{
			Steps: []resource.TestStep{
				{
					Config: hclStep(t, "resources/workspace_binding/with_expiry_date", 1, vars),
					ConfigPlanChecks: resource.ConfigPlanChecks{
						PreApply: []plancheck.PlanCheck{
							plancheck.ExpectResourceAction(resourceAddress, plancheck.ResourceActionCreate),
						},
					},
					ConfigStateChecks: append([]statecheck.StateCheck{
						statecheck.ExpectKnownValue(resourceAddress, tfjsonpath.New("metadata").AtMapKey("name"), knownvalue.StringExact(bindingName)),
						statecheck.ExpectKnownValue(resourceAddress, tfjsonpath.New("role_ref").AtMapKey("name"), knownvalue.StringExact("Workspace Member")),
						statecheck.ExpectKnownValue(resourceAddress, tfjsonpath.New("target_ref").AtMapKey("name"), knownvalue.StringExact("test-ws-"+suffix)),
						statecheck.ExpectKnownValue(resourceAddress, tfjsonpath.New("subject").AtMapKey("name"), knownvalue.StringExact(subjectName)),
						statecheck.ExpectKnownValue(resourceAddress, tfjsonpath.New("expiry_date"), knownvalue.StringExact("2099-12-31")),
					}, subjectStateChecks(resourceAddress, subjectChecks)...),
				},
				{
					ResourceName:    resourceAddress,
					ImportState:     true,
					ImportStateId:   bindingName,
					ImportStateKind: resource.ImportBlockWithID,
				},
			},
		})
	})

	// Omitting expiry_date leaves the Optional+Computed attribute unknown in the plan, which used to
	// fail the create with a "Received unknown value" conversion error (#267, #293).
	t.Run("without_expiry_date", func(t *testing.T) {
		suffix := acctest.RandString(8)
		vars := hclStepVars{"suffix": suffix, "subject_kind": subjectKind, "subject_name": subjectName}
		bindingName := "test-wb-" + suffix
		resourceAddress := "meshstack_workspace_" + subjectKind + "_binding.test"

		ApplyAndTest(t, resource.TestCase{
			Steps: []resource.TestStep{
				{
					Config: hclStep(t, "resources/workspace_binding/without_expiry_date", 1, vars),
					ConfigPlanChecks: resource.ConfigPlanChecks{
						PreApply: []plancheck.PlanCheck{
							plancheck.ExpectResourceAction(resourceAddress, plancheck.ResourceActionCreate),
						},
					},
					ConfigStateChecks: append([]statecheck.StateCheck{
						statecheck.ExpectKnownValue(resourceAddress, tfjsonpath.New("metadata").AtMapKey("name"), knownvalue.StringExact(bindingName)),
						statecheck.ExpectKnownValue(resourceAddress, tfjsonpath.New("subject").AtMapKey("name"), knownvalue.StringExact(subjectName)),
						statecheck.ExpectKnownValue(resourceAddress, tfjsonpath.New("expiry_date"), knownvalue.Null()),
					}, subjectStateChecks(resourceAddress, subjectChecks)...),
				},
				{
					ResourceName:    resourceAddress,
					ImportState:     true,
					ImportStateId:   bindingName,
					ImportStateKind: resource.ImportBlockWithID,
				},
			},
		})
	})
}
