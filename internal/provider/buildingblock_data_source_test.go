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

func TestAccBuildingBlockDataSource(t *testing.T) {
	if !IsMockClientTest() {
		t.Skip("Skipping: BB v1 resource has no wait_for_completion, BB run stays PENDING and blocks destroy")
	}

	bbConfig, buildingBlockAddr := testconfig.BBv1Tenant(t)

	var dataSourceAddr testconfig.Traversal
	config := testconfig.DataSource{Name: "buildingblock"}.Config(t).WithFirstBlock(
		testconfig.ExtractAddress(&dataSourceAddr),
		testconfig.Descend("metadata", "uuid")(testconfig.SetAddr(buildingBlockAddr, "metadata", "uuid"))).
		Join(bbConfig)

	ApplyAndTest(t, resource.TestCase{
		Steps: []resource.TestStep{
			{
				Config: config.String(),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(dataSourceAddr.String(), tfjsonpath.New("metadata").AtMapKey("uuid"), xknownvalue.NotEmptyString()),
					statecheck.ExpectKnownValue(dataSourceAddr.String(), tfjsonpath.New("spec").AtMapKey("display_name"), knownvalue.StringExact("my-buildingblock")),
				},
			},
		},
	})
}
