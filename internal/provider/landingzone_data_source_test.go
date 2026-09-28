package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/knownvalue"
	"github.com/hashicorp/terraform-plugin-testing/statecheck"
	"github.com/hashicorp/terraform-plugin-testing/tfjsonpath"

	"github.com/meshcloud/terraform-provider-meshstack/examples"
	"github.com/meshcloud/terraform-provider-meshstack/internal/provider/acctest/xknownvalue"
)

func TestAccLandingZoneDataSource(t *testing.T) {
	config := examples.JoinTestStepConfigs(
		examples.DataSource.TestStepConfig(t, "landingzone", 1),
		landingZoneStepConfig(t, 1),
	)

	ApplyAndTest(t, resource.TestCase{
		Steps: []resource.TestStep{
			{
				Config:          config,
				ConfigVariables: SuffixVariables(acctest.RandString(8)),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(landingZoneDataSourceAddr, tfjsonpath.New("metadata").AtMapKey("name"), xknownvalue.NotEmptyString()),
					statecheck.ExpectKnownValue(landingZoneDataSourceAddr, tfjsonpath.New("spec").AtMapKey("display_name"), knownvalue.StringExact("My Custom Landing Zone")),
				},
			},
		},
	})
}
