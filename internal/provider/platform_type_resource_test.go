package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/knownvalue"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
	"github.com/hashicorp/terraform-plugin-testing/statecheck"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	"github.com/hashicorp/terraform-plugin-testing/tfjsonpath"

	"github.com/meshcloud/terraform-provider-meshstack/internal/provider/acctest/testconfig"
	"github.com/meshcloud/terraform-provider-meshstack/internal/provider/acctest/xknownvalue"
)

func TestAccPlatformType(t *testing.T) {
	config, platformTypeAddr := testconfig.PlatformTypeAndWorkspace(t)
	resourceAddress := platformTypeAddr.String()

	// Use a random suffix to avoid state pollution from previous test runs.
	updateSuffix := acctest.RandString(8)
	updatedConfig := config.WithFirstBlock(
		testconfig.Descend("spec", "display_name")(testconfig.SetString("My Custom Platform Updated " + updateSuffix)),
	)

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
					statecheck.ExpectKnownValue(resourceAddress, tfjsonpath.New("metadata"), checkPlatformTypeMetadata()),
					statecheck.ExpectKnownValue(resourceAddress, tfjsonpath.New("spec").AtMapKey("display_name"), xknownvalue.KnownStringWithPrefix("My Custom Platform ")),
					statecheck.ExpectKnownValue(resourceAddress, tfjsonpath.New("status"), checkPlatformTypeStatus()),
					statecheck.ExpectKnownValue(resourceAddress, tfjsonpath.New("ref"), checkPlatformTypeRef()),
				},
			},
			{
				Config: updatedConfig.String(),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction(resourceAddress, plancheck.ResourceActionUpdate),
					},
				},
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(resourceAddress, tfjsonpath.New("spec").AtMapKey("display_name"), xknownvalue.KnownStringWithPrefix("My Custom Platform Updated")),
				},
			},
			{
				ImportState:     true,
				ImportStateKind: resource.ImportBlockWithID,
				ResourceName:    resourceAddress,
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

func checkPlatformTypeMetadata() knownvalue.Check {
	return xknownvalue.MapExact(map[string]knownvalue.Check{
		"name":               xknownvalue.NotEmptyString(),
		"owned_by_workspace": xknownvalue.NotEmptyString(),
		"uuid":               xknownvalue.NotEmptyString(),
	})
}

func checkPlatformTypeStatus() knownvalue.Check {
	return xknownvalue.MapExact(map[string]knownvalue.Check{
		"lifecycle": xknownvalue.MapExact(map[string]knownvalue.Check{
			"state": knownvalue.StringExact("ACTIVE"),
		}),
	})
}

func checkPlatformTypeRef() knownvalue.Check {
	return xknownvalue.MapExact(map[string]knownvalue.Check{
		"kind": knownvalue.StringExact("meshPlatformType"),
		"name": xknownvalue.NotEmptyString(),
	})
}
