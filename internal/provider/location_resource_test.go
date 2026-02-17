package provider

import (
	_ "embed"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/knownvalue"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
	"github.com/hashicorp/terraform-plugin-testing/statecheck"
	"github.com/hashicorp/terraform-plugin-testing/tfjsonpath"
	"github.com/stretchr/testify/assert"

	"github.com/meshcloud/terraform-provider-meshstack/examples"
	"github.com/meshcloud/terraform-provider-meshstack/internal/clientmock"
)

func TestAccLocation(t *testing.T) {
	runLocationResourceTestCase(t)
}

func TestLocation(t *testing.T) {
	// Run acceptance tests as unit tests with mock
	runLocationResourceTestCase(t, SetupMockClient(func(t *testing.T, testCase *resource.TestCase, mockClient clientmock.Client) {
		t.Helper()
		testCase.Steps[0].PostApplyFunc = func() {
			assert.Len(t, mockClient.Location.Store, 1)
		}
	}))
}

func runLocationResourceTestCase(t *testing.T, modifiers ...ResourceTestCaseModifier) {
	t.Helper()
	var resourceAddress examples.Identifier
	config := examples.Resource{Name: "location"}.Config().
		SingleResourceAddress(&resourceAddress)

	const resourceIdentifier = "my-location"

	testCase := resource.TestCase{
		Steps: []resource.TestStep{
			{
				Config: config.String(),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction(resourceAddress.String(), plancheck.ResourceActionCreate),
					},
				},
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(resourceAddress.String(), tfjsonpath.New("metadata"), checkLocationMetadata(resourceIdentifier)),
					statecheck.ExpectKnownValue(resourceAddress.String(), tfjsonpath.New("spec"), checkLocationSpec("My Cloud Location")),
					statecheck.ExpectKnownValue(resourceAddress.String(), tfjsonpath.New("status"), checkLocationStatus()),
					statecheck.ExpectKnownValue(resourceAddress.String(), tfjsonpath.New("ref"), checkLocationRef(resourceIdentifier)),
				},
			},
			{
				Config: config.ReplaceAll(`"My Cloud Location"`, `"My Updated Location"`).String(),
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
				ImportStateId:   resourceIdentifier,
				ResourceName:    resourceAddress.String(),
			},
		},
	}

	ResourceTestCaseModifiers(modifiers).ApplyAndTest(t, testCase)
}

func checkLocationMetadata(name string) knownvalue.Check {
	return knownvalue.MapExact(map[string]knownvalue.Check{
		"name":               knownvalue.StringExact(name),
		"owned_by_workspace": knownvalue.StringExact("my-workspace-identifier"),
		"uuid":               KnownValueNotEmptyString(),
	})
}

func checkLocationSpec(displayName string) knownvalue.Check {
	return knownvalue.MapExact(map[string]knownvalue.Check{
		"display_name": knownvalue.StringExact(displayName),
		"description":  knownvalue.StringExact("A location for managing cloud resources"),
	})
}

func checkLocationStatus() knownvalue.Check {
	return knownvalue.MapExact(map[string]knownvalue.Check{
		"is_public": knownvalue.Bool(false),
	})
}

func checkLocationRef(name string) knownvalue.Check {
	return knownvalue.MapExact(map[string]knownvalue.Check{
		"name": knownvalue.StringExact(name),
	})
}
