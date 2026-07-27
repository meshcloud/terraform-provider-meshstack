package provider

import (
	"fmt"
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

func TestAccWorkspaceTags(t *testing.T) {
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
