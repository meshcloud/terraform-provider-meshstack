package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/knownvalue"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
	"github.com/hashicorp/terraform-plugin-testing/statecheck"
	"github.com/hashicorp/terraform-plugin-testing/tfjsonpath"

	"github.com/meshcloud/terraform-provider-meshstack/internal/provider/acctest/testconfig"
	"github.com/meshcloud/terraform-provider-meshstack/internal/provider/acctest/xknownvalue"
)

func TestAccLocation(t *testing.T) {
	workspaceConfig, workspaceAddr := testconfig.Workspace(t)
	locationConfig, locationAddr, locationName := testconfig.Location(t, workspaceAddr)

	config := locationConfig.Join(workspaceConfig)

	updateConfig := config.WithFirstBlock(
		testconfig.Descend("spec", "display_name")(testconfig.SetString("My Updated Location")))

	ApplyAndTest(t, resource.TestCase{
		Steps: []resource.TestStep{
			{
				Config: config.String(),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction(locationAddr.String(), plancheck.ResourceActionCreate),
					},
				},
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(locationAddr.String(), tfjsonpath.New("metadata"), checkLocationMetadata(locationName)),
					statecheck.ExpectKnownValue(locationAddr.String(), tfjsonpath.New("spec"), checkLocationSpec("My Cloud Location")),
					statecheck.ExpectKnownValue(locationAddr.String(), tfjsonpath.New("status"), checkLocationStatus()),
					statecheck.ExpectKnownValue(locationAddr.String(), tfjsonpath.New("ref"), checkLocationRef(locationName)),
				},
			},
			{
				Config: updateConfig.String(),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction(locationAddr.String(), plancheck.ResourceActionUpdate),
					},
				},
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(locationAddr.String(), tfjsonpath.New("spec"), checkLocationSpec("My Updated Location")),
				},
			},
			{
				ImportState:     true,
				ImportStateKind: resource.ImportBlockWithID,
				ImportStateId:   locationName,
				ResourceName:    locationAddr.String(),
			},
		},
	})
}

func checkLocationMetadata(name string) knownvalue.Check {
	return xknownvalue.MapExact(map[string]knownvalue.Check{
		"name":               knownvalue.StringExact(name),
		"owned_by_workspace": xknownvalue.NotEmptyString(),
		"uuid":               xknownvalue.NotEmptyString(),
	})
}

func checkLocationSpec(displayName string) knownvalue.Check {
	return xknownvalue.MapExact(map[string]knownvalue.Check{
		"display_name": knownvalue.StringExact(displayName),
		"description":  knownvalue.StringExact("A location for managing cloud resources"),
	})
}

func checkLocationStatus() knownvalue.Check {
	return xknownvalue.MapExact(map[string]knownvalue.Check{
		"is_public": knownvalue.Bool(false),
	})
}

func checkLocationRef(name string) knownvalue.Check {
	return xknownvalue.MapExact(map[string]knownvalue.Check{
		"name": knownvalue.StringExact(name),
	})
}
