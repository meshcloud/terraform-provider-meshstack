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
	"github.com/meshcloud/terraform-provider-meshstack/internal/provider/acctest/xknownvalue"
)

// Address and tag-key prefixes of the blocks in
// examples/resources/meshstack_workspace_tags/*-test-*.tf.
const (
	workspaceTagsResourceAddr    = "meshstack_workspace_tags.example"
	workspaceTagsFirstKeyPrefix  = "test-key-wsts-first-"
	workspaceTagsSecondKeyPrefix = "test-key-wsts-second-"
)

// workspaceTagsStepConfig joins a workspace tags step file with the tag definitions it writes under
// and the untagged workspace it tags — variant 3, so this resource owns its tags.
func workspaceTagsStepConfig(t *testing.T, index int, supportNames ...string) string {
	t.Helper()
	return examples.JoinTestStepConfigs(
		examples.Resource.TestStepConfig(t, "workspace_tags", index, supportNames...),
		examples.Resource.TestStepConfig(t, "workspace", 3, "variables"),
	)
}

func TestAccWorkspaceTags(t *testing.T) {
	t.Run("declared_empty_value_list", func(t *testing.T) {
		suffix := acctest.RandString(8)

		ApplyAndTest(t, resource.TestCase{
			Steps: []resource.TestStep{
				{
					Config:          workspaceTagsStepConfig(t, 3, "prerequisites"),
					ConfigVariables: SuffixVariables(suffix),
					ConfigStateChecks: []statecheck.StateCheck{
						statecheck.ExpectKnownValue(workspaceTagsResourceAddr, tfjsonpath.New("spec").AtMapKey("tags"), knownvalue.MapExact(map[string]knownvalue.Check{
							workspaceTagsFirstKeyPrefix + suffix: knownvalue.ListSizeExact(0),
						})),
					},
					ConfigPlanChecks: resource.ConfigPlanChecks{
						PostApplyPostRefresh: []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()},
					},
				},
			},
		})
	})

	t.Run("removing_one_key_of_several", func(t *testing.T) {
		// The map is authoritative per key, not just as a whole: dropping one key of two must remove that
		// tag and leave the other untouched.
		suffix := acctest.RandString(8)
		vars := SuffixVariables(suffix)

		ApplyAndTest(t, resource.TestCase{
			Steps: []resource.TestStep{
				{
					Config:          workspaceTagsStepConfig(t, 4, "prerequisites", "second-tag-definition"),
					ConfigVariables: vars,
					ConfigStateChecks: []statecheck.StateCheck{
						statecheck.ExpectKnownValue(workspaceTagsResourceAddr, tfjsonpath.New("spec").AtMapKey("tags"), knownvalue.MapExact(map[string]knownvalue.Check{
							workspaceTagsFirstKeyPrefix + suffix:  knownvalue.ListExact([]knownvalue.Check{knownvalue.StringExact("first")}),
							workspaceTagsSecondKeyPrefix + suffix: knownvalue.ListExact([]knownvalue.Check{knownvalue.StringExact("second")}),
						})),
					},
				},
				{
					// Only the first key remains declared; the second tag definition stays so dropping the
					// key is what removes the tag, not the definition disappearing.
					Config: examples.JoinTestStepConfigs(
						workspaceTagsStepConfig(t, 1, "prerequisites"),
						examples.Resource.TestSupportConfigs(t, "workspace_tags", "second-tag-definition"),
					),
					ConfigVariables: vars,
					ConfigPlanChecks: resource.ConfigPlanChecks{
						PreApply: []plancheck.PlanCheck{
							plancheck.ExpectResourceAction(workspaceTagsResourceAddr, plancheck.ResourceActionUpdate),
						},
						PostApplyPostRefresh: []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()},
					},
					ConfigStateChecks: []statecheck.StateCheck{
						statecheck.ExpectKnownValue(workspaceTagsResourceAddr, tfjsonpath.New("spec").AtMapKey("tags"), knownvalue.MapExact(map[string]knownvalue.Check{
							workspaceTagsFirstKeyPrefix + suffix: knownvalue.ListExact([]knownvalue.Check{knownvalue.StringExact("12345")}),
						})),
					},
				},
			},
		})
	})

	t.Run("workspace_not_found", func(t *testing.T) {
		config := examples.JoinTestStepConfigs(
			examples.Resource.TestStepConfig(t, "workspace_tags", 5, "prerequisites"),
			examples.Resource.TestSupportConfigs(t, "workspace", "variables"),
		)

		ApplyAndTest(t, resource.TestCase{
			Steps: []resource.TestStep{
				{
					Config:          config,
					ConfigVariables: SuffixVariables(acctest.RandString(8)),
					ExpectError:     regexp.MustCompile(`Workspace .* not found`),
				},
			},
		})
	})

	vars := SuffixVariables(acctest.RandString(8))

	ApplyAndTest(t, resource.TestCase{
		Steps: []resource.TestStep{
			{
				Config:          workspaceTagsStepConfig(t, 1, "prerequisites"),
				ConfigVariables: vars,
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction(workspaceTagsResourceAddr, plancheck.ResourceActionCreate),
					},
				},
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(workspaceTagsResourceAddr, tfjsonpath.New("metadata").AtMapKey("workspace_identifier"), xknownvalue.NotEmptyString()),
					statecheck.ExpectKnownValue(workspaceTagsResourceAddr, tfjsonpath.New("spec").AtMapKey("tags"), knownvalue.MapSizeExact(1)),
				},
			},
			{
				Config:          workspaceTagsStepConfig(t, 2, "prerequisites"),
				ConfigVariables: vars,
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction(workspaceTagsResourceAddr, plancheck.ResourceActionUpdate),
					},
				},
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(workspaceTagsResourceAddr, tfjsonpath.New("spec").AtMapKey("tags"), knownvalue.MapSizeExact(0)),
				},
			},
			{
				// Restore the tag so the import below runs against a workspace that actually has tags.
				Config:          workspaceTagsStepConfig(t, 1, "prerequisites"),
				ConfigVariables: vars,
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction(workspaceTagsResourceAddr, plancheck.ResourceActionUpdate),
					},
				},
			},
			{
				// Read must pass the API's tags through when there is no prior state, rather than
				// reconcile against an empty tracked set and import nothing.
				ImportState:     true,
				ImportStateKind: resource.ImportBlockWithID,
				ConfigVariables: vars,
				ImportStateIdFunc: func(s *terraform.State) (string, error) {
					ws := s.RootModule().Resources[workspaceResourceAddr]
					if ws == nil {
						return "", fmt.Errorf("workspace resource not found: %s", workspaceResourceAddr)
					}
					return ws.Primary.Attributes["metadata.name"], nil
				},
				ResourceName: workspaceTagsResourceAddr,
			},
		},
	})
}
