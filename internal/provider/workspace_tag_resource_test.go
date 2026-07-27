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

func TestAccWorkspaceTag(t *testing.T) {
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
		},
	})
}
