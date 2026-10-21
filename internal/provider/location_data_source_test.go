package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/statecheck"
	"github.com/hashicorp/terraform-plugin-testing/tfjsonpath"

	"github.com/meshcloud/terraform-provider-meshstack/internal/provider/acctest/testconfig"
)

func TestAccLocationDataSource(t *testing.T) {
	workspaceConfig, workspaceAddr := testconfig.Workspace(t)
	locationConfig, locationAddr, locationName := testconfig.Location(t, workspaceAddr)

	var dataSourceAddress testconfig.Traversal
	config := testconfig.DataSource{Name: "location"}.Config(t).WithFirstBlock(
		testconfig.ExtractAddress(&dataSourceAddress),
		testconfig.Descend("metadata", "name")(testconfig.SetAddr(locationAddr, "metadata", "name")),
	).Join(locationConfig, workspaceConfig)

	ApplyAndTest(t, resource.TestCase{
		Steps: []resource.TestStep{
			{
				Config: config.String(),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(dataSourceAddress.String(), tfjsonpath.New("metadata"), checkLocationMetadata(locationName)),
					statecheck.ExpectKnownValue(dataSourceAddress.String(), tfjsonpath.New("spec"), checkLocationSpec("My Cloud Location")),
					statecheck.ExpectKnownValue(dataSourceAddress.String(), tfjsonpath.New("status"), checkLocationStatus()),
					statecheck.ExpectKnownValue(dataSourceAddress.String(), tfjsonpath.New("ref"), checkLocationRef(locationName)),
				},
			},
		},
	})
}
