package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/statecheck"
	"github.com/hashicorp/terraform-plugin-testing/tfjsonpath"
)

func TestAccLocationDataSource(t *testing.T) {
	suffix := acctest.RandString(8)
	vars := hclStepVars{"suffix": suffix}
	dataSourceAddress := "data.meshstack_location.example"
	locationName := "test-loc-" + suffix

	ApplyAndTest(t, resource.TestCase{
		Steps: []resource.TestStep{
			{
				Config: hclStep(t, "data-sources/location", 1, vars),
			},
			{
				Config: hclStep(t, "data-sources/location", 2, vars),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(dataSourceAddress, tfjsonpath.New("metadata"), checkLocationMetadata(locationName, "test-ws-"+suffix)),
					statecheck.ExpectKnownValue(dataSourceAddress, tfjsonpath.New("spec"), checkLocationSpec("My Cloud Location")),
					statecheck.ExpectKnownValue(dataSourceAddress, tfjsonpath.New("status"), checkLocationStatus()),
					statecheck.ExpectKnownValue(dataSourceAddress, tfjsonpath.New("ref"), checkLocationRef(locationName)),
				},
			},
		},
	})
}
