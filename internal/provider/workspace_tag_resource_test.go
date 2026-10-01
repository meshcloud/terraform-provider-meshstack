package provider

import (
	"fmt"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/knownvalue"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
	"github.com/hashicorp/terraform-plugin-testing/statecheck"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	"github.com/hashicorp/terraform-plugin-testing/tfjsonpath"

	"github.com/meshcloud/terraform-provider-meshstack/examples"
)

// Addresses and tag-key prefixes of the blocks in
// examples/resources/meshstack_workspace_tag/*-test-*.tf.
const (
	workspaceTagFirstAddr       = "meshstack_workspace_tag.first"
	workspaceTagSecondAddr      = "meshstack_workspace_tag.second"
	workspaceTagFirstKeyPrefix  = "test-key-wst-first-"
	workspaceTagSecondKeyPrefix = "test-key-wst-second-"
)

// workspaceTagStepConfig joins the named workspace tag step files with the tag definitions they
// write under and the untagged workspace they tag — variant 3, so these resources own its tags.
func workspaceTagStepConfig(t *testing.T, supportNames []string, indexes ...int) string {
	t.Helper()
	parts := make([]string, 0, len(indexes)+2)
	for _, index := range indexes {
		parts = append(parts, examples.Resource.TestStepConfig(t, "workspace_tag", index))
	}
	parts = append(parts,
		examples.Resource.TestSupportConfigs(t, "workspace_tag", supportNames...),
		examples.Resource.TestStepConfig(t, "workspace", 3, "variables"),
	)
	return examples.JoinTestStepConfigs(parts...)
}

func TestAccWorkspaceTag(t *testing.T) {
	t.Run("two_tags_on_one_workspace", func(t *testing.T) {
		// The resource's primary use case, and the "tags under other keys are read and written back
		// unchanged" claim in its description: each tag is a full read-modify-write of the workspace, so
		// the second write must preserve the first tag and deleting one must leave the other alone. Each
		// resource's Read only looks at its own key, so a clobbered tag surfaces as its resource dropping
		// out of state — hence the empty-plan checks.
		suffix := acctest.RandString(8)
		vars := NewVariablesWithSuffix(suffix)

		ApplyAndTest(t, resource.TestCase{
			Steps: []resource.TestStep{
				{
					Config:          workspaceTagStepConfig(t, []string{"prerequisites", "second-tag-definition"}, 1, 3),
					ConfigVariables: vars,
					ConfigStateChecks: []statecheck.StateCheck{
						statecheck.ExpectKnownValue(workspaceTagFirstAddr, tfjsonpath.New("metadata").AtMapKey("key"), knownvalue.StringExact(workspaceTagFirstKeyPrefix+suffix)),
						statecheck.ExpectKnownValue(workspaceTagFirstAddr, tfjsonpath.New("spec").AtMapKey("values"), knownvalue.ListExact([]knownvalue.Check{knownvalue.StringExact("12345")})),
						statecheck.ExpectKnownValue(workspaceTagSecondAddr, tfjsonpath.New("metadata").AtMapKey("key"), knownvalue.StringExact(workspaceTagSecondKeyPrefix+suffix)),
						statecheck.ExpectKnownValue(workspaceTagSecondAddr, tfjsonpath.New("spec").AtMapKey("values"), knownvalue.ListExact([]knownvalue.Check{knownvalue.StringExact("second")})),
					},
					ConfigPlanChecks: resource.ConfigPlanChecks{
						PostApplyPostRefresh: []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()},
					},
				},
				{
					// Removing the second tag must not take the first one with it: Delete rewrites the
					// workspace without its own key only.
					Config:          workspaceTagStepConfig(t, []string{"prerequisites"}, 1),
					ConfigVariables: vars,
					ConfigPlanChecks: resource.ConfigPlanChecks{
						PreApply: []plancheck.PlanCheck{
							plancheck.ExpectResourceAction(workspaceTagSecondAddr, plancheck.ResourceActionDestroy),
						},
						PostApplyPostRefresh: []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()},
					},
					ConfigStateChecks: []statecheck.StateCheck{
						statecheck.ExpectKnownValue(workspaceTagFirstAddr, tfjsonpath.New("spec").AtMapKey("values"), knownvalue.ListExact([]knownvalue.Check{knownvalue.StringExact("12345")})),
					},
				},
			},
		})
	})

	t.Run("workspace_not_found", func(t *testing.T) {
		// Step 4 names a workspace that does not exist and is applied without the workspace example's
		// step, so nothing creates it.
		config := examples.JoinTestStepConfigs(
			examples.Resource.TestStepConfig(t, "workspace_tag", 4),
			examples.Resource.TestSupportConfigs(t, "workspace_tag", "prerequisites"),
			examples.Resource.TestSupportConfigs(t, "workspace", "variables"),
		)

		ApplyAndTest(t, resource.TestCase{
			Steps: []resource.TestStep{
				{
					Config:          config,
					ConfigVariables: NewVariablesWithSuffix(acctest.RandString(8)),
					ExpectError:     regexp.MustCompile(`Workspace .* not found`),
				},
			},
		})
	})

	suffix := acctest.RandString(8)
	vars := NewVariablesWithSuffix(suffix)

	ApplyAndTest(t, resource.TestCase{
		Steps: []resource.TestStep{
			{
				Config:          workspaceTagStepConfig(t, []string{"prerequisites"}, 1),
				ConfigVariables: vars,
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction(workspaceTagFirstAddr, plancheck.ResourceActionCreate),
					},
				},
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(workspaceTagFirstAddr, tfjsonpath.New("metadata").AtMapKey("workspace_identifier"), knownvalue.StringExact("test-ws-"+suffix)),
					statecheck.ExpectKnownValue(workspaceTagFirstAddr, tfjsonpath.New("metadata").AtMapKey("key"), knownvalue.StringExact("test-key-wst-first-"+suffix)),
					statecheck.ExpectKnownValue(workspaceTagFirstAddr, tfjsonpath.New("spec").AtMapKey("values"), knownvalue.ListSizeExact(1)),
				},
			},
			{
				Config:          workspaceTagStepConfig(t, []string{"prerequisites"}, 2),
				ConfigVariables: vars,
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction(workspaceTagFirstAddr, plancheck.ResourceActionUpdate),
					},
				},
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(workspaceTagFirstAddr, tfjsonpath.New("spec").AtMapKey("values"), knownvalue.ListSizeExact(2)),
				},
			},
			{
				ImportState:     true,
				ImportStateKind: resource.ImportBlockWithID,
				ConfigVariables: vars,
				ImportStateIdFunc: func(s *terraform.State) (string, error) {
					ws := s.RootModule().Resources[workspaceResourceAddr]
					if ws == nil {
						return "", fmt.Errorf("workspace resource not found: %s", workspaceResourceAddr)
					}
					tag := s.RootModule().Resources[workspaceTagFirstAddr]
					if tag == nil {
						return "", fmt.Errorf("workspace tag resource not found: %s", workspaceTagFirstAddr)
					}
					return ws.Primary.Attributes["metadata.name"] + "." + tag.Primary.Attributes["metadata.key"], nil
				},
				ResourceName: workspaceTagFirstAddr,
			},
			{
				ImportState:     true,
				ResourceName:    workspaceTagFirstAddr,
				ImportStateId:   "no-dot",
				ConfigVariables: vars,
				ExpectError:     regexp.MustCompile(`Unexpected Import Identifier`),
			},
		},
	})
}
