package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/knownvalue"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
	"github.com/hashicorp/terraform-plugin-testing/statecheck"
	"github.com/hashicorp/terraform-plugin-testing/tfjsonpath"

	"github.com/meshcloud/terraform-provider-meshstack/examples"
)

// Addresses and names of the blocks in
// examples/resources/meshstack_workspace_{group,user}_binding/resource-test-*.tf. Step 1 carries the
// example's own binding name; step 2 builds one from the run's suffix.
const (
	workspaceGroupBindingResourceAddr = "meshstack_workspace_group_binding.example"
	workspaceUserBindingResourceAddr  = "meshstack_workspace_user_binding.example"
	workspaceBindingName              = "this-is-an-example"
	workspaceGroupBindingNamePrefix   = "test-wgb-"
	workspaceUserBindingNamePrefix    = "test-wub-"
)

// workspaceBindingStepConfig is a workspace binding example's step, with the workspace it targets.
func workspaceBindingStepConfig(t *testing.T, name string, index int) string {
	t.Helper()
	return examples.JoinTestStepConfigs(
		examples.Resource.TestStepConfig(t, name, index),
		examples.Resource.TestStepConfig(t, "workspace", 1, "variables", "prerequisites"),
	)
}

func TestAccWorkspaceGroupBinding(t *testing.T) {
	if !IsMockClientTest() {
		t.Skip("Skipping: requires user group 'my-user-group' in local meshStack")
	}

	t.Parallel()

	t.Run("with_expiry_date", func(t *testing.T) {
		suffix := acctest.RandString(8)
		vars := SuffixVariables(suffix)

		ApplyAndTest(t, resource.TestCase{
			Steps: []resource.TestStep{
				{
					Config:          workspaceBindingStepConfig(t, "workspace_group_binding", 1),
					ConfigVariables: vars,
					ConfigPlanChecks: resource.ConfigPlanChecks{
						PreApply: []plancheck.PlanCheck{
							plancheck.ExpectResourceAction(workspaceGroupBindingResourceAddr, plancheck.ResourceActionCreate),
						},
					},
					ConfigStateChecks: []statecheck.StateCheck{
						statecheck.ExpectKnownValue(workspaceGroupBindingResourceAddr, tfjsonpath.New("metadata").AtMapKey("name"), knownvalue.StringExact(workspaceBindingName)),
						statecheck.ExpectKnownValue(workspaceGroupBindingResourceAddr, tfjsonpath.New("role_ref").AtMapKey("name"), knownvalue.StringExact("Workspace Member")),
						statecheck.ExpectKnownValue(workspaceGroupBindingResourceAddr, tfjsonpath.New("target_ref").AtMapKey("name"), knownvalue.StringExact("test-ws-"+suffix)),
						statecheck.ExpectKnownValue(workspaceGroupBindingResourceAddr, tfjsonpath.New("subject").AtMapKey("name"), knownvalue.StringExact("my-user-group")),
						statecheck.ExpectKnownValue(workspaceGroupBindingResourceAddr, tfjsonpath.New("expiry_date"), knownvalue.StringExact("2026-12-31")),
					},
				},
				{
					ResourceName:    workspaceGroupBindingResourceAddr,
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
		bindingName := workspaceGroupBindingNamePrefix + suffix

		ApplyAndTest(t, resource.TestCase{
			Steps: []resource.TestStep{
				{
					Config:          workspaceBindingStepConfig(t, "workspace_group_binding", 2),
					ConfigVariables: vars,
					ConfigPlanChecks: resource.ConfigPlanChecks{
						PreApply: []plancheck.PlanCheck{
							plancheck.ExpectResourceAction(workspaceGroupBindingResourceAddr, plancheck.ResourceActionCreate),
						},
					},
					ConfigStateChecks: []statecheck.StateCheck{
						statecheck.ExpectKnownValue(workspaceGroupBindingResourceAddr, tfjsonpath.New("metadata").AtMapKey("name"), knownvalue.StringExact(bindingName)),
						statecheck.ExpectKnownValue(workspaceGroupBindingResourceAddr, tfjsonpath.New("subject").AtMapKey("name"), knownvalue.StringExact("my-user-group")),
						statecheck.ExpectKnownValue(workspaceGroupBindingResourceAddr, tfjsonpath.New("expiry_date"), knownvalue.Null()),
					},
				},
				{
					ResourceName:    workspaceGroupBindingResourceAddr,
					ImportState:     true,
					ImportStateId:   bindingName,
					ImportStateKind: resource.ImportBlockWithID,
					ConfigVariables: vars,
				},
			},
		})
	})
}
