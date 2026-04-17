package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/knownvalue"
	"github.com/hashicorp/terraform-plugin-testing/statecheck"
	"github.com/hashicorp/terraform-plugin-testing/tfjsonpath"

	"github.com/meshcloud/terraform-provider-meshstack/internal/provider/acctest/testconfig"
)

func TestAccTagDefinitionDataSource(t *testing.T) {
	resourceConfig, _, tagKey := testconfig.TagDefinition(t, "meshProject")

	var dataSourceAddr testconfig.Traversal
	config := testconfig.DataSource{Name: "tag_definition"}.Config(t).WithFirstBlock(
		testconfig.ExtractAddress(&dataSourceAddr),
		testconfig.Descend("name")(testconfig.SetString("meshProject."+tagKey)),
	).Join(resourceConfig)

	ApplyAndTest(t, resource.TestCase{
		Steps: []resource.TestStep{
			{
				Config: config.String(),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(dataSourceAddr.String(), tfjsonpath.New("name"), knownvalue.StringExact("meshProject."+tagKey)),
				},
			},
		},
	})
}
