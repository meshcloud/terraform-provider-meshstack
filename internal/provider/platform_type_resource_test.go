package provider

import (
	"fmt"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/knownvalue"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
	"github.com/hashicorp/terraform-plugin-testing/statecheck"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	"github.com/hashicorp/terraform-plugin-testing/tfjsonpath"

	"github.com/meshcloud/terraform-provider-meshstack/examples"
	"github.com/meshcloud/terraform-provider-meshstack/internal/provider/acctest/xknownvalue"
)

// Addresses of the blocks in examples/{resources,data-sources}/meshstack_platform_type/*-test-*.tf.
const (
	platformTypeResourceAddr   = "meshstack_platform_type.example"
	platformTypeDataSourceAddr = "data.meshstack_platform_type.example"
)

// platformTypeStepConfig is the platform type example's step, with the workspace that owns it.
func platformTypeStepConfig(t *testing.T, index int) string {
	t.Helper()
	return examples.JoinTestStepConfigs(
		examples.Resource.TestStepConfig(t, "platform_type", index),
		examples.Resource.TestStepConfig(t, "workspace", 1, "variables", "prerequisites"),
	)
}

func platformTypeName(suffix string) string {
	return "CUSTOM-PT-" + strings.ToUpper(suffix)
}

func TestAccPlatformType(t *testing.T) {
	suffix := acctest.RandString(8)
	vars := SuffixVariables(suffix)

	ApplyAndTest(t, resource.TestCase{
		Steps: []resource.TestStep{
			{
				Config:          platformTypeStepConfig(t, 1),
				ConfigVariables: vars,
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction(platformTypeResourceAddr, plancheck.ResourceActionCreate),
					},
				},
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(platformTypeResourceAddr, tfjsonpath.New("metadata"), checkPlatformTypeMetadata(suffix)),
					statecheck.ExpectKnownValue(platformTypeResourceAddr, tfjsonpath.New("spec").AtMapKey("display_name"), xknownvalue.KnownStringWithPrefix("My Custom Platform ")),
					statecheck.ExpectKnownValue(platformTypeResourceAddr, tfjsonpath.New("status"), checkPlatformTypeStatus()),
					statecheck.ExpectKnownValue(platformTypeResourceAddr, tfjsonpath.New("ref"), checkPlatformTypeRef(suffix)),
				},
			},
			{
				Config:          platformTypeStepConfig(t, 2),
				ConfigVariables: vars,
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction(platformTypeResourceAddr, plancheck.ResourceActionUpdate),
					},
				},
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(platformTypeResourceAddr, tfjsonpath.New("spec").AtMapKey("display_name"), xknownvalue.KnownStringWithPrefix("My Custom Platform Updated")),
				},
			},
			{
				ImportState:     true,
				ImportStateKind: resource.ImportBlockWithID,
				ResourceName:    platformTypeResourceAddr,
				ConfigVariables: vars,
				ImportStateIdFunc: func(s *terraform.State) (string, error) {
					rs := s.RootModule().Resources[platformTypeResourceAddr]
					if rs == nil {
						return "", fmt.Errorf("resource not found: %s", platformTypeResourceAddr)
					}
					return rs.Primary.Attributes["metadata.name"], nil
				},
			},
		},
	})
}

func checkPlatformTypeMetadata(suffix string) knownvalue.Check {
	return xknownvalue.MapExact(map[string]knownvalue.Check{
		"name":               knownvalue.StringExact(platformTypeName(suffix)),
		"owned_by_workspace": knownvalue.StringExact("test-ws-" + suffix),
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

func checkPlatformTypeRef(suffix string) knownvalue.Check {
	return xknownvalue.MapExact(map[string]knownvalue.Check{
		"kind": knownvalue.StringExact("meshPlatformType"),
		"name": knownvalue.StringExact(platformTypeName(suffix)),
	})
}
