package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/statecheck"
	"github.com/hashicorp/terraform-plugin-testing/tfjsonpath"

	"github.com/meshcloud/terraform-provider-meshstack/internal/provider/acctest/testconfig"
	"github.com/meshcloud/terraform-provider-meshstack/internal/provider/acctest/xknownvalue"
)

func TestAccTagDefinitionResource(t *testing.T) {
	suffix := acctest.RandString(8)

	config := testconfig.Resource{Name: "tag_definition"}.Config(t).WithFirstBlock(t,
		testconfig.Traverse(t, "spec", "key")(testconfig.SetString("test-key-"+suffix)),
		testconfig.Traverse(t, "spec", "display_name")(testconfig.SetString("Example "+suffix)),
	)

	ApplyAndTest(t, resource.TestCase{
		Steps: []resource.TestStep{
			{
				Config: config.String(),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue("meshstack_tag_definition.example", tfjsonpath.New("spec").AtMapKey("display_name"), xknownvalue.KnownStringWithPrefix("Example")),
				},
			},
		},
	})
}
