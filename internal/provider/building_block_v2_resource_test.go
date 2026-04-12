package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/knownvalue"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
	"github.com/hashicorp/terraform-plugin-testing/statecheck"
	"github.com/hashicorp/terraform-plugin-testing/tfjsonpath"

	"github.com/meshcloud/terraform-provider-meshstack/internal/provider/acctest/testconfig"
	"github.com/meshcloud/terraform-provider-meshstack/internal/provider/acctest/xknownvalue"
)

func TestAccBuildingBlockV2(t *testing.T) {
	t.Parallel()

	t.Run("01_workspace", func(t *testing.T) {
		config, bbAddr := testconfig.BuildBBv2WorkspaceConfig(t)

		ApplyAndTest(t, resource.TestCase{
			Steps: []resource.TestStep{
				{
					Config: config.String(),
					ConfigPlanChecks: resource.ConfigPlanChecks{
						PreApply: []plancheck.PlanCheck{
							plancheck.ExpectResourceAction(bbAddr.String(), plancheck.ResourceActionCreate),
						},
					},
					ConfigStateChecks: bbv2StateChecks(bbAddr, "my-workspace-building-block"),
				},
			},
		})
	})

	t.Run("02_tenant", func(t *testing.T) {
		config, bbAddr := testconfig.BuildBBv2TenantConfig(t)

		ApplyAndTest(t, resource.TestCase{
			Steps: []resource.TestStep{
				{
					Config: config.String(),
					ConfigPlanChecks: resource.ConfigPlanChecks{
						PreApply: []plancheck.PlanCheck{
							plancheck.ExpectResourceAction(bbAddr.String(), plancheck.ResourceActionCreate),
						},
					},
					ConfigStateChecks: bbv2StateChecks(bbAddr, "my-tenant-building-block"),
				},
			},
		})
	})
}

func bbv2StateChecks(bbAddr testconfig.Traversal, displayName string) []statecheck.StateCheck {
	return []statecheck.StateCheck{
		statecheck.ExpectKnownValue(bbAddr.String(), tfjsonpath.New("metadata").AtMapKey("uuid"), xknownvalue.NotEmptyString()),
		statecheck.ExpectKnownValue(bbAddr.String(), tfjsonpath.New("spec").AtMapKey("display_name"), knownvalue.StringExact(displayName)),
		statecheck.ExpectKnownValue(bbAddr.String(), tfjsonpath.New("spec").AtMapKey("inputs").AtMapKey("name").AtMapKey("value_string"), knownvalue.StringExact("my-name")),
		statecheck.ExpectKnownValue(bbAddr.String(), tfjsonpath.New("spec").AtMapKey("inputs").AtMapKey("size").AtMapKey("value_int"), knownvalue.Int64Exact(16)),
		statecheck.ExpectKnownValue(bbAddr.String(), tfjsonpath.New("spec").AtMapKey("inputs").AtMapKey("environment").AtMapKey("value_single_select"), knownvalue.StringExact("dev")),
		statecheck.ExpectKnownValue(bbAddr.String(), tfjsonpath.New("status").AtMapKey("status"), knownvalue.StringExact("SUCCEEDED")),
	}
}
