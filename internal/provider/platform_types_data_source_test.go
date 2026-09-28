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

func TestAccPlatformTypesDataSource(t *testing.T) {
	// Create a platform type first so the mock store is non-empty; the step file's depends_on keeps
	// the listing ordered after it.
	config := examples.JoinTestStepConfigs(
		examples.DataSource.TestStepConfig(t, "platform_types", 1),
		platformTypeStepConfig(t, 1),
	)

	ApplyAndTest(t, resource.TestCase{
		Steps: []resource.TestStep{
			{
				Config:          config,
				ConfigVariables: SuffixVariables(acctest.RandString(8)),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue("data.meshstack_platform_types.all", tfjsonpath.New("platform_types"), knownvalue.NotNull()),
				},
			},
		},
	})
}
