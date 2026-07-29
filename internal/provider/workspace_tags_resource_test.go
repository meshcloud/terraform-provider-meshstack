package provider

import (
	"fmt"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/knownvalue"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
	"github.com/hashicorp/terraform-plugin-testing/statecheck"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	"github.com/hashicorp/terraform-plugin-testing/tfjsonpath"

	"github.com/meshcloud/terraform-provider-meshstack/client"
	"github.com/meshcloud/terraform-provider-meshstack/internal/provider/acctest/testconfig"
	"github.com/meshcloud/terraform-provider-meshstack/internal/provider/acctest/xknownvalue"
)

func TestAccWorkspaceTags(t *testing.T) {
	t.Run("declared_empty_value_list", func(t *testing.T) {
		// A declared key with no values must converge. The API returns an entry for every defined tag
		// property — an empty list when unset — and reconcileTags mirrors tracked keys verbatim, so the
		// key stays in state instead of being dropped on every refresh.
		tagConfig, tagDefinitionAddr, tagKey := testconfig.TagDefinition(t, client.MeshObjectKind.Workspace)
		workspaceConfig, workspaceAddr := testconfig.WorkspaceWithoutTags(t)

		var workspaceTagsAddr testconfig.Traversal
		config := testconfig.Resource{Name: "workspace_tags"}.Config(t).WithFirstBlock(
			testconfig.ExtractAddress(&workspaceTagsAddr),
			testconfig.Descend("metadata", "workspace_identifier")(testconfig.SetAddr(workspaceAddr, "metadata", "name")),
			testconfig.Descend("spec", "tags")(testconfig.SetRawExpr(`{(%s) = []}`, tagDefinitionAddr.Join("spec", "key"))),
		).Join(tagConfig, workspaceConfig)

		ApplyAndTest(t, resource.TestCase{
			Steps: []resource.TestStep{
				{
					Config: config.String(),
					ConfigStateChecks: []statecheck.StateCheck{
						statecheck.ExpectKnownValue(workspaceTagsAddr.String(), tfjsonpath.New("spec").AtMapKey("tags"), knownvalue.MapExact(map[string]knownvalue.Check{
							tagKey: knownvalue.ListSizeExact(0),
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
		firstTagConfig, firstTagAddr, firstKey := testconfig.TagDefinition(t, client.MeshObjectKind.Workspace)
		secondTagConfig, secondTagAddr, secondKey := testconfig.TagDefinition(t, client.MeshObjectKind.Workspace)
		workspaceConfig, workspaceAddr := testconfig.WorkspaceWithoutTags(t)

		var workspaceTagsAddr testconfig.Traversal
		bothKeys := testconfig.Resource{Name: "workspace_tags"}.Config(t).WithFirstBlock(
			testconfig.ExtractAddress(&workspaceTagsAddr),
			testconfig.Descend("metadata", "workspace_identifier")(testconfig.SetAddr(workspaceAddr, "metadata", "name")),
			testconfig.Descend("spec", "tags")(testconfig.SetRawExpr(
				`{(%s) = ["first"], (%s) = ["second"]}`,
				firstTagAddr.Join("spec", "key"),
				secondTagAddr.Join("spec", "key"),
			)),
		).Join(firstTagConfig, secondTagConfig, workspaceConfig)

		onlyFirstKey := bothKeys.WithFirstBlock(
			testconfig.Descend("spec", "tags")(testconfig.SetRawExpr(`{(%s) = ["first"]}`, firstTagAddr.Join("spec", "key"))),
		)

		ApplyAndTest(t, resource.TestCase{
			Steps: []resource.TestStep{
				{
					Config: bothKeys.String(),
					ConfigStateChecks: []statecheck.StateCheck{
						statecheck.ExpectKnownValue(workspaceTagsAddr.String(), tfjsonpath.New("spec").AtMapKey("tags"), knownvalue.MapExact(map[string]knownvalue.Check{
							firstKey:  knownvalue.ListExact([]knownvalue.Check{knownvalue.StringExact("first")}),
							secondKey: knownvalue.ListExact([]knownvalue.Check{knownvalue.StringExact("second")}),
						})),
					},
				},
				{
					Config: onlyFirstKey.String(),
					ConfigPlanChecks: resource.ConfigPlanChecks{
						PreApply: []plancheck.PlanCheck{
							plancheck.ExpectResourceAction(workspaceTagsAddr.String(), plancheck.ResourceActionUpdate),
						},
						PostApplyPostRefresh: []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()},
					},
					ConfigStateChecks: []statecheck.StateCheck{
						statecheck.ExpectKnownValue(workspaceTagsAddr.String(), tfjsonpath.New("spec").AtMapKey("tags"), knownvalue.MapExact(map[string]knownvalue.Check{
							firstKey: knownvalue.ListExact([]knownvalue.Check{knownvalue.StringExact("first")}),
						})),
					},
				},
			},
		})
	})

	t.Run("workspace_not_found", func(t *testing.T) {
		// The workspace config is deliberately not joined: only its generated identifier is reused, so the
		// tag resource points at a workspace that does not exist.
		_, workspaceAddr := testconfig.WorkspaceWithoutTags(t)
		workspaceTagsConfig, _ := testconfig.WorkspaceTags(t, workspaceAddr)
		config := workspaceTagsConfig.WithFirstBlock(
			testconfig.Descend("metadata", "workspace_identifier")(testconfig.SetString("does-not-exist-workspace")),
		)

		ApplyAndTest(t, resource.TestCase{
			Steps: []resource.TestStep{
				{
					Config:      config.String(),
					ExpectError: regexp.MustCompile(`Workspace .* not found`),
				},
			},
		})
	})

	workspaceConfig, workspaceAddr := testconfig.WorkspaceWithoutTags(t)
	workspaceTagsConfig, workspaceTagsAddr := testconfig.WorkspaceTags(t, workspaceAddr)
	config := workspaceTagsConfig.Join(workspaceConfig)

	updateConfig := config.WithFirstBlock(
		testconfig.Descend("spec", "tags")(testconfig.SetRawExpr(`{}`)),
	)

	ApplyAndTest(t, resource.TestCase{
		Steps: []resource.TestStep{
			{
				Config: config.String(),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction(workspaceTagsAddr.String(), plancheck.ResourceActionCreate),
					},
				},
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(workspaceTagsAddr.String(), tfjsonpath.New("metadata").AtMapKey("workspace_identifier"), xknownvalue.NotEmptyString()),
					statecheck.ExpectKnownValue(workspaceTagsAddr.String(), tfjsonpath.New("spec").AtMapKey("tags"), knownvalue.MapSizeExact(1)),
				},
			},
			{
				Config: updateConfig.String(),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction(workspaceTagsAddr.String(), plancheck.ResourceActionUpdate),
					},
				},
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(workspaceTagsAddr.String(), tfjsonpath.New("spec").AtMapKey("tags"), knownvalue.MapSizeExact(0)),
				},
			},
			{
				// Restore the tag so the import below runs against a workspace that actually has tags.
				Config: config.String(),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction(workspaceTagsAddr.String(), plancheck.ResourceActionUpdate),
					},
				},
			},
			{
				// Read must pass the API's tags through when there is no prior state, rather than
				// reconcile against an empty tracked set and import nothing.
				ImportState:     true,
				ImportStateKind: resource.ImportBlockWithID,
				ImportStateIdFunc: func(s *terraform.State) (string, error) {
					ws := s.RootModule().Resources[workspaceAddr.String()]
					if ws == nil {
						return "", fmt.Errorf("workspace resource not found: %s", workspaceAddr.String())
					}
					return ws.Primary.Attributes["metadata.name"], nil
				},
				ResourceName: workspaceTagsAddr.String(),
			},
		},
	})
}
