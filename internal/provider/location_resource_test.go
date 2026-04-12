package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/knownvalue"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
	"github.com/hashicorp/terraform-plugin-testing/statecheck"
	"github.com/hashicorp/terraform-plugin-testing/tfjsonpath"

	"github.com/meshcloud/terraform-provider-meshstack/internal/provider/acctest/testconfig"
	"github.com/meshcloud/terraform-provider-meshstack/internal/provider/acctest/xknownvalue"
)

func TestAccLocation(t *testing.T) {
	workspaceConfig, workspaceAddr := testconfig.BuildWorkspaceConfig(t)

	locationName := "my-location-" + acctest.RandString(32)
	locationConfig := testconfig.Resource{Name: "location"}.Config(t)
	var resourceAddress testconfig.Traversal
	locationConfig = locationConfig.WithFirstBlock(t, testconfig.OwnedByWorkspace(t, workspaceAddr))
	locationConfig = locationConfig.WithFirstBlock(t,
		testconfig.ExtractIdentifier(&resourceAddress),
		testconfig.Traverse(t, "metadata", "name")(testconfig.SetString(locationName)))

	config := locationConfig.Join(workspaceConfig)

	updateConfig := config.WithFirstBlock(t,
		testconfig.Traverse(t, "spec", "display_name")(testconfig.SetString("My Updated Location")))

	ApplyAndTest(t, resource.TestCase{
		Steps: []resource.TestStep{
			{
				Config: config.String(),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction(resourceAddress.String(), plancheck.ResourceActionCreate),
					},
				},
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(resourceAddress.String(), tfjsonpath.New("metadata"), checkLocationMetadata(locationName)),
					statecheck.ExpectKnownValue(resourceAddress.String(), tfjsonpath.New("spec"), checkLocationSpec("My Cloud Location")),
					statecheck.ExpectKnownValue(resourceAddress.String(), tfjsonpath.New("status"), checkLocationStatus()),
					statecheck.ExpectKnownValue(resourceAddress.String(), tfjsonpath.New("ref"), checkLocationRef(locationName)),
				},
			},
			{
				Config: updateConfig.String(),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction(resourceAddress.String(), plancheck.ResourceActionUpdate),
					},
				},
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(resourceAddress.String(), tfjsonpath.New("spec"), checkLocationSpec("My Updated Location")),
				},
			},
			{
				ImportState:     true,
				ImportStateKind: resource.ImportBlockWithID,
				ImportStateId:   locationName,
				ResourceName:    resourceAddress.String(),
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
