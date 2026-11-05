package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/knownvalue"
	"github.com/hashicorp/terraform-plugin-testing/statecheck"
	"github.com/hashicorp/terraform-plugin-testing/tfjsonpath"
)

func TestAccTagDefinitionResource(t *testing.T) {
	vars := hclStepVars{"suffix": acctest.RandString(8)}
	tagDefinitionAddr := "meshstack_tag_definition.example"

	ApplyAndTest(t, resource.TestCase{
		Steps: []resource.TestStep{
			{
				Config: hclStep(t, "resources/tag_definition", 1, vars),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(tagDefinitionAddr, tfjsonpath.New("spec").AtMapKey("display_name"), knownvalue.StringExact("Example")),
				},
			},
		},
	})
}

func TestAccTagDefinitionResource_SingleSelectWithLocalOptions(t *testing.T) {
	vars := hclStepVars{"suffix": acctest.RandString(8)}
	addr := "meshstack_tag_definition.single_select_local"

	ApplyAndTest(t, resource.TestCase{
		Steps: []resource.TestStep{
			{
				Config: hclStep(t, "resources/tag_definition/single_select_with_local_options", 1, vars),
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
	vars := hclStepVars{"suffix": acctest.RandString(8)}
	addr := "meshstack_tag_definition.single_select_numbers"

	ApplyAndTest(t, resource.TestCase{
		Steps: []resource.TestStep{
			{
				Config: hclStep(t, "resources/tag_definition/options_coerce_numbers_to_strings", 1, vars),
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
