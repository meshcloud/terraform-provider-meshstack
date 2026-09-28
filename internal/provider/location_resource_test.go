package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/knownvalue"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
	"github.com/hashicorp/terraform-plugin-testing/statecheck"
	"github.com/hashicorp/terraform-plugin-testing/tfjsonpath"

	"github.com/meshcloud/terraform-provider-meshstack/examples"
	"github.com/meshcloud/terraform-provider-meshstack/internal/provider/acctest/xknownvalue"
)

// Address and name prefix of the block in examples/resources/meshstack_location/resource-test-*.tf.
const (
	locationResourceAddr = "meshstack_location.example"
	locationNamePrefix   = "my-location-"
)

func TestAccLocation(t *testing.T) {
	suffix := acctest.RandString(8)
	vars := SuffixVariables(suffix)
	locationName := locationNamePrefix + suffix

	stepConfig := func(index int) string {
		return examples.JoinTestStepConfigs(
			examples.Resource.TestStepConfig(t, "location", index),
			examples.Resource.TestStepConfig(t, "workspace", 1, "variables", "prerequisites"),
		)
	}

	ApplyAndTest(t, resource.TestCase{
		Steps: []resource.TestStep{
			{
				Config:          stepConfig(1),
				ConfigVariables: vars,
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction(locationResourceAddr, plancheck.ResourceActionCreate),
					},
				},
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(locationResourceAddr, tfjsonpath.New("metadata"), checkLocationMetadata(locationName)),
					statecheck.ExpectKnownValue(locationResourceAddr, tfjsonpath.New("spec"), checkLocationSpec("My Cloud Location")),
					statecheck.ExpectKnownValue(locationResourceAddr, tfjsonpath.New("status"), checkLocationStatus()),
					statecheck.ExpectKnownValue(locationResourceAddr, tfjsonpath.New("ref"), checkLocationRef(locationName)),
				},
			},
			{
				Config:          stepConfig(2),
				ConfigVariables: vars,
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction(locationResourceAddr, plancheck.ResourceActionUpdate),
					},
				},
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(locationResourceAddr, tfjsonpath.New("spec"), checkLocationSpec("My Updated Location")),
				},
			},
			{
				ImportState:     true,
				ImportStateKind: resource.ImportBlockWithID,
				ImportStateId:   locationName,
				ConfigVariables: vars,
				ResourceName:    locationResourceAddr,
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
		"kind": knownvalue.StringExact("meshLocation"),
		"name": knownvalue.StringExact(name),
	})
}
