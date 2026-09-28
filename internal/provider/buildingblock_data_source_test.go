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

func TestAccBuildingblockDataSource(t *testing.T) {
	if !IsMockClientTest() {
		t.Skip("Skipping: BB v1 resource has no wait_for_completion, BB run stays PENDING and blocks destroy")
	}

	config := examples.JoinTestStepConfigs(
		examples.DataSource.TestStepConfig(t, "buildingblock", 1),
		buildingBlockV1StepConfig(t),
	)

	ApplyAndTest(t, resource.TestCase{
		Steps: []resource.TestStep{
			{
				Config:          config,
				ConfigVariables: SuffixVariables(acctest.RandString(8)),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(buildingBlockV1DataSourceAddr, tfjsonpath.New("metadata").AtMapKey("uuid"), xknownvalue.NotEmptyString()),
					statecheck.ExpectKnownValue(buildingBlockV1DataSourceAddr, tfjsonpath.New("spec").AtMapKey("display_name"), knownvalue.StringExact("my-buildingblock")),
				},
			},
		},
	})
}
