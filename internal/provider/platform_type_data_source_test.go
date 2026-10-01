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

func TestAccPlatformTypeDataSource(t *testing.T) {
	config := examples.JoinTestStepConfigs(
		examples.DataSource.TestStepConfig(t, "platform_type", 1),
		platformTypeStepConfig(t, 1),
	)

	suffix := acctest.RandString(8)

	ApplyAndTest(t, resource.TestCase{
		Steps: []resource.TestStep{
			{
				Config:          config,
				ConfigVariables: NewVariablesWithSuffix(suffix),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(platformTypeDataSourceAddr, tfjsonpath.New("metadata"), checkPlatformTypeMetadata(suffix)),
					statecheck.ExpectKnownValue(platformTypeDataSourceAddr, tfjsonpath.New("spec").AtMapKey("display_name"), knownvalue.StringExact("My Custom Platform "+suffix)),
					statecheck.ExpectKnownValue(platformTypeDataSourceAddr, tfjsonpath.New("status"), checkPlatformTypeStatus()),
					statecheck.ExpectKnownValue(platformTypeDataSourceAddr, tfjsonpath.New("ref"), checkPlatformTypeRef(suffix)),
				},
			},
		},
	})
}
