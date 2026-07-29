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

	"github.com/meshcloud/terraform-provider-meshstack/internal/provider/acctest/testconfig"
	"github.com/meshcloud/terraform-provider-meshstack/internal/provider/acctest/xknownvalue"
)

func TestAccWorkspaceTag(t *testing.T) {
	t.Run("two_tags_on_one_workspace", func(t *testing.T) {
		// The resource's primary use case, and the "tags under other keys are read and written back
		// unchanged" claim in its description: each tag is a full read-modify-write of the workspace, so
		// the second write must preserve the first tag and deleting one must leave the other alone. Each
		// resource's Read only looks at its own key, so a clobbered tag surfaces as its resource dropping
		// out of state — hence the empty-plan checks.
		workspaceConfig, workspaceAddr := testconfig.WorkspaceWithoutTags(t)
		firstConfig, firstAddr, firstKey := testconfig.WorkspaceTag(t, workspaceAddr)
		secondConfig, secondAddr, secondKey := testconfig.WorkspaceTag(t, workspaceAddr)

		// depends_on serializes the two read-modify-write cycles. Without it Terraform applies them in
		// parallel and one tag is silently lost — the race documented in workspaceTagCaveats.
		secondConfig = secondConfig.WithFirstBlock(
			testconfig.Descend("spec", "values")(testconfig.SetRawExpr(`["second"]`)),
			testconfig.Descend("depends_on")(testconfig.SetRawExpr("[%s]", firstAddr)),
		)

		bothTags := firstConfig.Join(secondConfig, workspaceConfig)
		onlyFirstTag := firstConfig.Join(workspaceConfig)

		ApplyAndTest(t, resource.TestCase{
			Steps: []resource.TestStep{
				{
					Config: bothTags.String(),
					ConfigStateChecks: []statecheck.StateCheck{
						statecheck.ExpectKnownValue(firstAddr.String(), tfjsonpath.New("metadata").AtMapKey("key"), knownvalue.StringExact(firstKey)),
						statecheck.ExpectKnownValue(firstAddr.String(), tfjsonpath.New("spec").AtMapKey("values"), knownvalue.ListExact([]knownvalue.Check{knownvalue.StringExact("12345")})),
						statecheck.ExpectKnownValue(secondAddr.String(), tfjsonpath.New("metadata").AtMapKey("key"), knownvalue.StringExact(secondKey)),
						statecheck.ExpectKnownValue(secondAddr.String(), tfjsonpath.New("spec").AtMapKey("values"), knownvalue.ListExact([]knownvalue.Check{knownvalue.StringExact("second")})),
					},
					ConfigPlanChecks: resource.ConfigPlanChecks{
						PostApplyPostRefresh: []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()},
					},
				},
				{
					// Removing the second tag must not take the first one with it: Delete rewrites the
					// workspace without its own key only.
					Config: onlyFirstTag.String(),
					ConfigPlanChecks: resource.ConfigPlanChecks{
						PreApply: []plancheck.PlanCheck{
							plancheck.ExpectResourceAction(secondAddr.String(), plancheck.ResourceActionDestroy),
						},
						PostApplyPostRefresh: []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()},
					},
					ConfigStateChecks: []statecheck.StateCheck{
						statecheck.ExpectKnownValue(firstAddr.String(), tfjsonpath.New("spec").AtMapKey("values"), knownvalue.ListExact([]knownvalue.Check{knownvalue.StringExact("12345")})),
					},
				},
			},
		})
	})

	t.Run("workspace_not_found", func(t *testing.T) {
		// This resource cannot create the workspace it tags, so a missing one must be a clear error
		// rather than a nil-deref or a silent no-op.
		// The workspace config is deliberately not joined: only its generated identifier is reused, so the
		// tag resource points at a workspace that does not exist.
		_, workspaceAddr := testconfig.WorkspaceWithoutTags(t)
		workspaceTagConfig, _, _ := testconfig.WorkspaceTag(t, workspaceAddr)
		config := workspaceTagConfig.WithFirstBlock(
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
	workspaceTagConfig, workspaceTagAddr, _ := testconfig.WorkspaceTag(t, workspaceAddr)
	config := workspaceTagConfig.Join(workspaceConfig)

	updateConfig := config.WithFirstBlock(
		testconfig.Descend("spec", "values")(testconfig.SetRawExpr(`["12345", "67890"]`)),
	)

	ApplyAndTest(t, resource.TestCase{
		Steps: []resource.TestStep{
			{
				Config: config.String(),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction(workspaceTagAddr.String(), plancheck.ResourceActionCreate),
					},
				},
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(workspaceTagAddr.String(), tfjsonpath.New("metadata").AtMapKey("workspace_identifier"), xknownvalue.NotEmptyString()),
					statecheck.ExpectKnownValue(workspaceTagAddr.String(), tfjsonpath.New("metadata").AtMapKey("key"), xknownvalue.NotEmptyString()),
					statecheck.ExpectKnownValue(workspaceTagAddr.String(), tfjsonpath.New("spec").AtMapKey("values"), knownvalue.ListSizeExact(1)),
				},
			},
			{
				Config: updateConfig.String(),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction(workspaceTagAddr.String(), plancheck.ResourceActionUpdate),
					},
				},
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(workspaceTagAddr.String(), tfjsonpath.New("spec").AtMapKey("values"), knownvalue.ListSizeExact(2)),
				},
			},
			{
				ImportState:     true,
				ImportStateKind: resource.ImportBlockWithID,
				ImportStateIdFunc: func(s *terraform.State) (string, error) {
					ws := s.RootModule().Resources[workspaceAddr.String()]
					if ws == nil {
						return "", fmt.Errorf("workspace resource not found: %s", workspaceAddr.String())
					}
					tag := s.RootModule().Resources[workspaceTagAddr.String()]
					if tag == nil {
						return "", fmt.Errorf("workspace tag resource not found: %s", workspaceTagAddr.String())
					}
					return ws.Primary.Attributes["metadata.name"] + "." + tag.Primary.Attributes["metadata.key"], nil
				},
				ResourceName: workspaceTagAddr.String(),
			},
			{
				ImportState:   true,
				ResourceName:  workspaceTagAddr.String(),
				ImportStateId: "no-dot",
				ExpectError:   regexp.MustCompile(`Unexpected Import Identifier`),
			},
		},
	})
}
