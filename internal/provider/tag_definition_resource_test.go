package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
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
