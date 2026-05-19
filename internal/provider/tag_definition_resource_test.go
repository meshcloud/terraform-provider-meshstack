package provider

import (
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/knownvalue"
	"github.com/hashicorp/terraform-plugin-testing/statecheck"
	"github.com/hashicorp/terraform-plugin-testing/tfjsonpath"

	"github.com/meshcloud/terraform-provider-meshstack/internal/provider/acctest/testconfig"
	"github.com/meshcloud/terraform-provider-meshstack/internal/provider/acctest/xknownvalue"
)

func TestAccTagDefinitionResource(t *testing.T) {
	config, tagDefinitionAddr, _ := testconfig.TagDefinition(t, "meshProject")

	ApplyAndTest(t, resource.TestCase{
		Steps: []resource.TestStep{
			{
				Config: config.String(),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(tagDefinitionAddr.String(), tfjsonpath.New("spec").AtMapKey("display_name"), xknownvalue.KnownStringWithPrefix("Example")),
				},
			},
		},
	})
}

func TestAccTagDefinitionResource_SingleSelectWithLocalOptions(t *testing.T) {
	keySuffix := acctest.RandString(8)
	addr := "meshstack_tag_definition.single_select_local"

	config := `
locals {
  tag_options = ["option_a", "option_b", "option_c"]
}

resource "meshstack_tag_definition" "single_select_local" {
  spec = {
    target_kind  = "meshProject"
    key          = "test-ss-local-` + keySuffix + `"
    display_name = "Test Single Select Local"
    value_type = {
      single_select = {
        options = local.tag_options
      }
    }
  }
}
`

	ApplyAndTest(t, resource.TestCase{
		Steps: []resource.TestStep{
			{
				Config: config,
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(addr, tfjsonpath.New("spec").AtMapKey("value_type").AtMapKey("single_select").AtMapKey("options"), knownvalue.ListExact([]knownvalue.Check{
						knownvalue.StringExact("option_a"),
						knownvalue.StringExact("option_b"),
						knownvalue.StringExact("option_c"),
					})),
				},
			},
		},
	})
}

// TestTagDefinitionResource_OptionsCoerceNumbersToStrings verifies that Terraform's
// type system automatically coerces number literals to strings in a list(string)
// context — options = [1, 2, 3] is accepted and stored as ["1", "2", "3"].
// This coercion happens at the HCL evaluation layer before provider code is reached
// There's nothing we can do about it, but we pin this (undesired) behavior here in the test.
func TestTagDefinitionResource_OptionsCoerceNumbersToStrings(t *testing.T) {
	keySuffix := acctest.RandString(8)
	addr := "meshstack_tag_definition.single_select_numbers"

	config := `
resource "meshstack_tag_definition" "single_select_numbers" {
  spec = {
    target_kind  = "meshProject"
    key          = "test-ss-numbers-` + keySuffix + `"
    display_name = "Test Single Select Numbers"
    value_type = {
      single_select = {
        options = [1, 2, 3]
      }
    }
  }
}
`

	ApplyAndTest(t, resource.TestCase{
		Steps: []resource.TestStep{
			{
				Config: config,
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(addr, tfjsonpath.New("spec").AtMapKey("value_type").AtMapKey("single_select").AtMapKey("options"), knownvalue.ListExact([]knownvalue.Check{
						knownvalue.StringExact("1"),
						knownvalue.StringExact("2"),
						knownvalue.StringExact("3"),
					})),
				},
			},
		},
	})
}

// TestTagDefinitionResource_OptionsRejectObjectElements verifies that the schema
// enforces list(string) element type for options — an object element cannot be
// coerced to string and must be rejected.
func TestTagDefinitionResource_OptionsRejectObjectElements(t *testing.T) {
	keySuffix := acctest.RandString(8)

	config := `
resource "meshstack_tag_definition" "invalid_options" {
  spec = {
    target_kind  = "meshProject"
    key          = "test-invalid-elem-` + keySuffix + `"
    display_name = "Test Invalid Element Type"
    value_type = {
      single_select = {
        options = [{ not_a_string = true }]
      }
    }
  }
}
`

	ApplyAndTest(t, resource.TestCase{
		Steps: []resource.TestStep{
			{
				Config:      config,
				ExpectError: regexp.MustCompile(`Incorrect attribute value type|Invalid value for`),
			},
		},
	})
}
