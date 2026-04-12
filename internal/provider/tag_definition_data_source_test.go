package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/knownvalue"
	"github.com/hashicorp/terraform-plugin-testing/statecheck"
	"github.com/hashicorp/terraform-plugin-testing/tfjsonpath"

	"github.com/meshcloud/terraform-provider-meshstack/internal/provider/acctest/testconfig"
)

func TestAccTagDefinitionDataSource(t *testing.T) {
	suffix := acctest.RandString(8)

	resourceConfig := testconfig.Resource{Name: "tag_definition"}.Config(t)
	resourceConfig = resourceConfig.WithFirstBlock(t,
		testconfig.Traverse(t, "spec", "key")(testconfig.SetString("test-key-"+suffix)),
		testconfig.Traverse(t, "spec", "display_name")(testconfig.SetString("Example "+suffix)),
	)

	dataSourceConfig := testconfig.DataSource{Name: "tag_definition"}.Config(t)
	dataSourceConfig = dataSourceConfig.WithFirstBlock(t,
		testconfig.Traverse(t, "name")(testconfig.SetString("meshProject.test-key-"+suffix)),
	)

	config := dataSourceConfig.Join(resourceConfig)

	ApplyAndTest(t, resource.TestCase{
		Steps: []resource.TestStep{
			{
				Config: config.String(),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue("data.meshstack_tag_definition.example", tfjsonpath.New("name"), knownvalue.StringExact("meshProject.test-key-"+suffix)),
				},
			},
		},
	})
}
