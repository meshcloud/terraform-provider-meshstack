package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/knownvalue"
	"github.com/hashicorp/terraform-plugin-testing/statecheck"
	"github.com/hashicorp/terraform-plugin-testing/tfjsonpath"
)

func TestAccLandingZoneDataSource(t *testing.T) {
	suffix := acctest.RandString(8)
	vars := platformTypeStepVars(suffix)
	dsAddress := "data.meshstack_landingzone.example"

	ApplyAndTest(t, resource.TestCase{
		Steps: []resource.TestStep{
			{
				Config: hclStep(t, "data-sources/landingzone", 1, vars),
			},
			{
				Config: hclStep(t, "data-sources/landingzone", 2, vars),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(dsAddress, tfjsonpath.New("metadata").AtMapKey("name"), knownvalue.StringExact("test-lz-"+suffix)),
					statecheck.ExpectKnownValue(dsAddress, tfjsonpath.New("spec").AtMapKey("display_name"), knownvalue.StringExact("My Custom Landing Zone")),
				},
			},
		},
	})
}
