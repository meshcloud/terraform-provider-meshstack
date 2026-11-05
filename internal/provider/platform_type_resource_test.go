package provider

import (
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/knownvalue"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
	"github.com/hashicorp/terraform-plugin-testing/statecheck"
	"github.com/hashicorp/terraform-plugin-testing/tfjsonpath"

	"github.com/meshcloud/terraform-provider-meshstack/internal/provider/acctest/xknownvalue"
)

func TestAccPlatformType(t *testing.T) {
	suffix := acctest.RandString(8)
	vars := platformTypeStepVars(suffix)
	resourceAddress := "meshstack_platform_type.example"
	platformTypeName := "TEST-PT-" + strings.ToUpper(suffix)

	ApplyAndTest(t, resource.TestCase{
		Steps: []resource.TestStep{
			{
				Config: hclStep(t, "resources/platform_type", 1, vars),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction(resourceAddress, plancheck.ResourceActionCreate),
					},
				},
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(resourceAddress, tfjsonpath.New("metadata"), checkPlatformTypeMetadata(platformTypeName, "test-ws-"+suffix)),
					statecheck.ExpectKnownValue(resourceAddress, tfjsonpath.New("spec").AtMapKey("display_name"), knownvalue.StringExact("My Custom Platform "+suffix)),
					statecheck.ExpectKnownValue(resourceAddress, tfjsonpath.New("status"), checkPlatformTypeStatus()),
					statecheck.ExpectKnownValue(resourceAddress, tfjsonpath.New("ref"), checkPlatformTypeRef(platformTypeName)),
				},
			},
			{
				Config: hclStep(t, "resources/platform_type", 2, vars),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction(resourceAddress, plancheck.ResourceActionUpdate),
					},
				},
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(resourceAddress, tfjsonpath.New("spec").AtMapKey("display_name"), knownvalue.StringExact("My Custom Platform Updated "+suffix)),
				},
			},
			{
				ImportState:     true,
				ImportStateKind: resource.ImportBlockWithID,
				ResourceName:    resourceAddress,
				ImportStateId:   platformTypeName,
			},
		},
	})
}

func platformTypeStepVars(suffix string) hclStepVars {
	return hclStepVars{"suffix": suffix, "upper_suffix": strings.ToUpper(suffix)}
}

func checkPlatformTypeMetadata(name, workspace string) knownvalue.Check {
	return xknownvalue.MapExact(map[string]knownvalue.Check{
		"name":               knownvalue.StringExact(name),
		"owned_by_workspace": knownvalue.StringExact(workspace),
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

func checkPlatformTypeRef(name string) knownvalue.Check {
	return xknownvalue.MapExact(map[string]knownvalue.Check{
		"kind": knownvalue.StringExact("meshPlatformType"),
		"name": knownvalue.StringExact(name),
	})
}
