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
	if !IsMockClientTest() {
		t.Skip("Skipping: requires user 'user@meshcloud.io' in local meshStack")
	}

	t.Parallel()

	t.Run("with_expiry_date", func(t *testing.T) {
		suffix := acctest.RandString(8)
		vars := SuffixVariables(suffix)

		ApplyAndTest(t, resource.TestCase{
			Steps: []resource.TestStep{
				{
					Config:          workspaceBindingStepConfig(t, "workspace_user_binding", 1),
					ConfigVariables: vars,
					ConfigPlanChecks: resource.ConfigPlanChecks{
						PreApply: []plancheck.PlanCheck{
							plancheck.ExpectResourceAction(workspaceUserBindingResourceAddr, plancheck.ResourceActionCreate),
						},
					},
					ConfigStateChecks: []statecheck.StateCheck{
						statecheck.ExpectKnownValue(workspaceUserBindingResourceAddr, tfjsonpath.New("metadata").AtMapKey("name"), knownvalue.StringExact(workspaceBindingName)),
						statecheck.ExpectKnownValue(workspaceUserBindingResourceAddr, tfjsonpath.New("role_ref").AtMapKey("name"), knownvalue.StringExact("Workspace Member")),
						statecheck.ExpectKnownValue(workspaceUserBindingResourceAddr, tfjsonpath.New("target_ref").AtMapKey("name"), knownvalue.StringExact("test-ws-"+suffix)),
						statecheck.ExpectKnownValue(workspaceUserBindingResourceAddr, tfjsonpath.New("subject").AtMapKey("name"), knownvalue.StringExact("user@meshcloud.io")),
						statecheck.ExpectKnownValue(workspaceUserBindingResourceAddr, tfjsonpath.New("expiry_date"), knownvalue.StringExact("2026-12-31")),
					},
				},
				{
					ResourceName:    workspaceUserBindingResourceAddr,
					ImportState:     true,
					ImportStateId:   workspaceBindingName,
					ImportStateKind: resource.ImportBlockWithID,
					ConfigVariables: vars,
				},
			},
		})
	})

	// Omitting expiry_date leaves the Optional+Computed attribute unknown in the plan, which used to
	// fail the create with a "Received unknown value" conversion error (#267, #293).
	t.Run("without_expiry_date", func(t *testing.T) {
		suffix := acctest.RandString(8)
		vars := SuffixVariables(suffix)
		bindingName := workspaceUserBindingNamePrefix + suffix

		ApplyAndTest(t, resource.TestCase{
			Steps: []resource.TestStep{
				{
					Config:          workspaceBindingStepConfig(t, "workspace_user_binding", 2),
					ConfigVariables: vars,
					ConfigPlanChecks: resource.ConfigPlanChecks{
						PreApply: []plancheck.PlanCheck{
							plancheck.ExpectResourceAction(workspaceUserBindingResourceAddr, plancheck.ResourceActionCreate),
						},
					},
					ConfigStateChecks: []statecheck.StateCheck{
						statecheck.ExpectKnownValue(workspaceUserBindingResourceAddr, tfjsonpath.New("metadata").AtMapKey("name"), knownvalue.StringExact(bindingName)),
						statecheck.ExpectKnownValue(workspaceUserBindingResourceAddr, tfjsonpath.New("subject").AtMapKey("name"), knownvalue.StringExact("user@meshcloud.io")),
						statecheck.ExpectKnownValue(workspaceUserBindingResourceAddr, tfjsonpath.New("expiry_date"), knownvalue.Null()),
					},
				},
				{
					ResourceName:    workspaceUserBindingResourceAddr,
					ImportState:     true,
					ImportStateId:   bindingName,
					ImportStateKind: resource.ImportBlockWithID,
					ConfigVariables: vars,
				},
			},
		})
	})
}
