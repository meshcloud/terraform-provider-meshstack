package provider

import (
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
