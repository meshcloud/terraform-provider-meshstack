package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/knownvalue"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
	"github.com/hashicorp/terraform-plugin-testing/statecheck"
	"github.com/hashicorp/terraform-plugin-testing/tfjsonpath"

	"github.com/meshcloud/terraform-provider-meshstack/internal/provider/acctest/xknownvalue"
)

func TestAccLocation(t *testing.T) {
	suffix := acctest.RandString(8)
	vars := hclStepVars{"suffix": suffix}
	locationAddr := "meshstack_location.example"
	locationName := "test-loc-" + suffix

	ApplyAndTest(t, resource.TestCase{
		Steps: []resource.TestStep{
			{
				Config: hclStep(t, "resources/location", 1, vars),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction(locationAddr, plancheck.ResourceActionCreate),
					},
				},
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(locationAddr, tfjsonpath.New("metadata"), checkLocationMetadata(locationName, "test-ws-"+suffix)),
					statecheck.ExpectKnownValue(locationAddr, tfjsonpath.New("spec"), checkLocationSpec("My Cloud Location")),
					statecheck.ExpectKnownValue(locationAddr, tfjsonpath.New("status"), checkLocationStatus()),
					statecheck.ExpectKnownValue(locationAddr, tfjsonpath.New("ref"), checkLocationRef(locationName)),
				},
			},
			{
				Config: hclStep(t, "resources/location", 2, vars),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction(locationAddr, plancheck.ResourceActionUpdate),
					},
				},
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(locationAddr, tfjsonpath.New("spec"), checkLocationSpec("My Updated Location")),
				},
			},
			{
				ImportState:     true,
				ImportStateKind: resource.ImportBlockWithID,
				ImportStateId:   locationName,
				ResourceName:    locationAddr,
			},
		},
	})
}

func checkLocationMetadata(name, workspace string) knownvalue.Check {
	return xknownvalue.MapExact(map[string]knownvalue.Check{
		"name":               knownvalue.StringExact(name),
		"owned_by_workspace": knownvalue.StringExact(workspace),
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
		"kind": knownvalue.StringExact("meshLocation"),
		"name": knownvalue.StringExact(name),
	})
}
