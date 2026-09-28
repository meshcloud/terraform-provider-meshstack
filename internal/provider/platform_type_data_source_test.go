package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/statecheck"
	"github.com/hashicorp/terraform-plugin-testing/tfjsonpath"

	"github.com/meshcloud/terraform-provider-meshstack/examples"
	"github.com/meshcloud/terraform-provider-meshstack/internal/provider/acctest/xknownvalue"
)

func TestAccPlatformTypeDataSource(t *testing.T) {
	config := examples.JoinTestStepConfigs(
		examples.DataSource.TestStepConfig(t, "platform_type", 1),
		platformTypeStepConfig(t, 1),
	)

	ApplyAndTest(t, resource.TestCase{
		Steps: []resource.TestStep{
			{
				Config:          config,
				ConfigVariables: SuffixVariables(acctest.RandString(8)),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(platformTypeDataSourceAddr, tfjsonpath.New("metadata"), checkPlatformTypeMetadata()),
					statecheck.ExpectKnownValue(platformTypeDataSourceAddr, tfjsonpath.New("spec").AtMapKey("display_name"), xknownvalue.NotEmptyString()),
					statecheck.ExpectKnownValue(platformTypeDataSourceAddr, tfjsonpath.New("status"), checkPlatformTypeStatus()),
					statecheck.ExpectKnownValue(platformTypeDataSourceAddr, tfjsonpath.New("ref"), checkPlatformTypeRef()),
				},
			},
		},
	})
}
