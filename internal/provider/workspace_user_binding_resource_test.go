package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/knownvalue"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
	"github.com/hashicorp/terraform-plugin-testing/statecheck"
	"github.com/hashicorp/terraform-plugin-testing/tfjsonpath"

	"github.com/meshcloud/terraform-provider-meshstack/internal/provider/acctest/testconfig"
	"github.com/meshcloud/terraform-provider-meshstack/internal/provider/acctest/xknownvalue"
)

func TestAccWorkspaceUserBinding(t *testing.T) {
	if !IsMockClientTest() {
		t.Skip("Skipping: requires user 'user@meshcloud.io' in local meshStack")
	}

	t.Parallel()

	t.Run("with_expiry_date", func(t *testing.T) {
		workspaceConfig, workspaceAddr := testconfig.Workspace(t)

		var resourceAddress testconfig.Traversal
		config := testconfig.Resource{Name: "workspace_user_binding"}.Config(t).WithFirstBlock(
			testconfig.ExtractAddress(&resourceAddress),
			testconfig.Descend("target_ref", "name")(testconfig.SetAddr(workspaceAddr, "metadata", "name"))).
			Join(workspaceConfig)

		ApplyAndTest(t, resource.TestCase{
			Steps: []resource.TestStep{
				{
					Config: config.String(),
					ConfigPlanChecks: resource.ConfigPlanChecks{
						PreApply: []plancheck.PlanCheck{
							plancheck.ExpectResourceAction(resourceAddress.String(), plancheck.ResourceActionCreate),
						},
					},
					ConfigStateChecks: []statecheck.StateCheck{
						statecheck.ExpectKnownValue(resourceAddress.String(), tfjsonpath.New("metadata").AtMapKey("name"), knownvalue.StringExact("this-is-an-example")),
						statecheck.ExpectKnownValue(resourceAddress.String(), tfjsonpath.New("role_ref").AtMapKey("name"), knownvalue.StringExact("Workspace Member")),
						statecheck.ExpectKnownValue(resourceAddress.String(), tfjsonpath.New("target_ref").AtMapKey("name"), xknownvalue.NotEmptyString()),
						statecheck.ExpectKnownValue(resourceAddress.String(), tfjsonpath.New("subject").AtMapKey("name"), knownvalue.StringExact("user@meshcloud.io")),
						statecheck.ExpectKnownValue(resourceAddress.String(), tfjsonpath.New("expiry_date"), knownvalue.StringExact("2026-12-31")),
					},
				},
				{
					ResourceName:    resourceAddress.String(),
					ImportState:     true,
					ImportStateId:   "this-is-an-example",
					ImportStateKind: resource.ImportBlockWithID,
				},
			},
		})
	})

	// Omitting expiry_date leaves the Optional+Computed attribute unknown in the plan, which used to
	// fail the create with a "Received unknown value" conversion error (#267, #293).
	t.Run("without_expiry_date", func(t *testing.T) {
		workspaceConfig, workspaceAddr := testconfig.Workspace(t)
		bindingName := "test-wub-" + acctest.RandString(8)

		var resourceAddress testconfig.Traversal
		config := testconfig.Resource{Name: "workspace_user_binding"}.Config(t).WithFirstBlock(
			testconfig.ExtractAddress(&resourceAddress),
			testconfig.Descend("metadata", "name")(testconfig.SetString(bindingName)),
			testconfig.Descend("expiry_date")(testconfig.RemoveKey()),
			testconfig.Descend("target_ref", "name")(testconfig.SetAddr(workspaceAddr, "metadata", "name"))).
			Join(workspaceConfig)

		ApplyAndTest(t, resource.TestCase{
			Steps: []resource.TestStep{
				{
					Config: config.String(),
					ConfigPlanChecks: resource.ConfigPlanChecks{
						PreApply: []plancheck.PlanCheck{
							plancheck.ExpectResourceAction(resourceAddress.String(), plancheck.ResourceActionCreate),
						},
					},
					ConfigStateChecks: []statecheck.StateCheck{
						statecheck.ExpectKnownValue(resourceAddress.String(), tfjsonpath.New("metadata").AtMapKey("name"), knownvalue.StringExact(bindingName)),
						statecheck.ExpectKnownValue(resourceAddress.String(), tfjsonpath.New("subject").AtMapKey("name"), knownvalue.StringExact("user@meshcloud.io")),
						statecheck.ExpectKnownValue(resourceAddress.String(), tfjsonpath.New("expiry_date"), knownvalue.Null()),
					},
				},
				{
					ResourceName:    resourceAddress.String(),
					ImportState:     true,
					ImportStateId:   bindingName,
					ImportStateKind: resource.ImportBlockWithID,
				},
			},
		})
	})
}
