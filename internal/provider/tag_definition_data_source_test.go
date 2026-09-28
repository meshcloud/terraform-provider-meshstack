package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/knownvalue"
	"github.com/hashicorp/terraform-plugin-testing/statecheck"
	"github.com/hashicorp/terraform-plugin-testing/tfjsonpath"

	"github.com/meshcloud/terraform-provider-meshstack/examples"
)

const tagDefinitionDataSourceAddr = "data.meshstack_tag_definition.example"

func TestAccTagDefinitionDataSource(t *testing.T) {
	suffix := acctest.RandString(8)

	config := examples.JoinTestStepConfigs(
		examples.DataSource.TestStepConfig(t, "tag_definition", 1),
		examples.Resource.TestStepConfig(t, "tag_definition", 1, "variables"),
	)

	ApplyAndTest(t, resource.TestCase{
		Steps: []resource.TestStep{
			{
				Config:          config,
				ConfigVariables: SuffixVariables(suffix),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(tagDefinitionDataSourceAddr, tfjsonpath.New("name"), knownvalue.StringExact("meshProject."+projectTagKeyPrefix+suffix)),
				},
			},
		},
	})
}
