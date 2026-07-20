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

// TestAccLandingZoneBuildingBlockRefRequiresUuid asserts the plan-time validator rejects a
// building block ref object that is provided without a uuid (an assigned computed `.ref`, whose
// uuid is unknown at plan time, stays allowed — see TestAccBuildingBlock/04_tenant_moved_from_v1).
func TestAccLandingZoneBuildingBlockRefRequiresUuid(t *testing.T) {
	config, _ := testconfig.LandingZoneAndWorkspace(t)

	badConfig := config.WithFirstBlock(
		testconfig.Descend("spec", "mandatory_building_block_refs")(
			testconfig.SetRawExpr(`[{ kind = "meshBuildingBlockDefinition" }]`)))

	ApplyAndTest(t, resource.TestCase{
		Steps: []resource.TestStep{
			{
				Config:      badConfig.String(),
				PlanOnly:    true,
				ExpectError: regexp.MustCompile(`(?s)uuid.*must be specified when`),
			},
		},
	})
}

func TestAccLandingZone(t *testing.T) {
	config, landingZoneAddr := testconfig.LandingZoneAndWorkspace(t)
	resourceAddress := landingZoneAddr.String()

	updateConfig := config.WithFirstBlock(
		testconfig.Descend("spec", "display_name")(testconfig.SetString("Updated Landing Zone")))

	ApplyAndTest(t, resource.TestCase{
		Steps: []resource.TestStep{
			{
				Config: config.String(),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction(resourceAddress, plancheck.ResourceActionCreate),
					},
				},
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(resourceAddress, tfjsonpath.New("metadata").AtMapKey("owned_by_workspace"), xknownvalue.NotEmptyString()),
					statecheck.ExpectKnownValue(resourceAddress, tfjsonpath.New("spec").AtMapKey("display_name"), knownvalue.StringExact("My Custom Landing Zone")),
					statecheck.ExpectKnownValue(resourceAddress, tfjsonpath.New("ref").AtMapKey("kind"), knownvalue.StringExact("meshLandingZone")),
					statecheck.ExpectKnownValue(resourceAddress, tfjsonpath.New("ref").AtMapKey("name"), xknownvalue.NotEmptyString()),
				},
			},
			{
				Config: updateConfig.String(),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction(resourceAddress, plancheck.ResourceActionUpdate),
					},
				},
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(resourceAddress, tfjsonpath.New("spec").AtMapKey("display_name"), knownvalue.StringExact("Updated Landing Zone")),
				},
			},
			{
				ResourceName:    resourceAddress,
				ImportState:     true,
				ImportStateKind: resource.ImportBlockWithID,
				ImportStateIdFunc: func(s *terraform.State) (string, error) {
					rs := s.RootModule().Resources[resourceAddress]
					if rs == nil {
						return "", fmt.Errorf("resource not found: %s", resourceAddress)
					}
					return rs.Primary.Attributes["metadata.name"], nil
				},
			},
		},
	})
}
