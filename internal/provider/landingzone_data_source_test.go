package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/knownvalue"
	"github.com/hashicorp/terraform-plugin-testing/statecheck"
	"github.com/hashicorp/terraform-plugin-testing/tfjsonpath"

	"github.com/meshcloud/terraform-provider-meshstack/internal/provider/acctest/testconfig"
	"github.com/meshcloud/terraform-provider-meshstack/internal/provider/acctest/xknownvalue"
)

func TestAccLandingZoneDataSource(t *testing.T) {
	landingZoneConfig, landingZoneAddr := testconfig.LandingZoneAndWorkspace(t)

	dsAddress := testconfig.Traversal{"data.meshstack_landingzone", "example"}
	config := testconfig.DataSource{Name: "landingzone"}.Config(t).WithFirstBlock(
		testconfig.Descend("metadata", "name")(testconfig.SetAddr(landingZoneAddr, "metadata", "name"))).
		Join(landingZoneConfig)

	ApplyAndTest(t, resource.TestCase{
		Steps: []resource.TestStep{
			{
				Config: config.String(),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(dsAddress.String(), tfjsonpath.New("metadata").AtMapKey("name"), xknownvalue.NotEmptyString()),
					statecheck.ExpectKnownValue(dsAddress.String(), tfjsonpath.New("spec").AtMapKey("display_name"), knownvalue.StringExact("My Custom Landing Zone")),
				},
			},
		},
	})
}
