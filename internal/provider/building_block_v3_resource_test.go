package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/knownvalue"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
	"github.com/hashicorp/terraform-plugin-testing/statecheck"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	"github.com/hashicorp/terraform-plugin-testing/tfjsonpath"

	"github.com/meshcloud/terraform-provider-meshstack/internal/provider/acctest/testconfig"
	"github.com/meshcloud/terraform-provider-meshstack/internal/provider/acctest/xknownvalue"
)

func TestAccBuildingBlockV3(t *testing.T) {
	t.Parallel()

	t.Run("01_workspace", func(t *testing.T) {
		config, bbAddr := testconfig.BuildBBv3WorkspaceConfig(t)
		updatedInputsConfig := config.WithFirstBlock(t,
			testconfig.Traverse(t, "spec", "inputs", "environment", "value")(testconfig.SetString("staging")),
		)
		retriggerConfig := updatedInputsConfig.WithFirstBlock(t,
			testconfig.Traverse(t, "retrigger_run")(testconfig.SetString("force-run-1")),
		)

		ApplyAndTest(t, resource.TestCase{
			Steps: []resource.TestStep{
				{
					Config: config.String(),
					ConfigPlanChecks: resource.ConfigPlanChecks{
						PreApply: []plancheck.PlanCheck{
							plancheck.ExpectResourceAction(bbAddr.String(), plancheck.ResourceActionCreate),
						},
					},
					ConfigStateChecks: bbv3StateChecks(bbAddr, "my-workspace-building-block"),
				},
				{
					ImportState:     true,
					ImportStateKind: resource.ImportBlockWithID,
					ImportStateIdFunc: func(s *terraform.State) (string, error) {
						rs := s.RootModule().Resources[bbAddr.String()]
						if rs == nil {
							return "", fmt.Errorf("resource not found: %s", bbAddr.String())
						}
						return rs.Primary.Attributes["metadata.uuid"], nil
					},
					ResourceName: bbAddr.String(),
				},
				{
					Config: updatedInputsConfig.String(),
					ConfigPlanChecks: resource.ConfigPlanChecks{
						PreApply: []plancheck.PlanCheck{
							plancheck.ExpectResourceAction(bbAddr.String(), plancheck.ResourceActionUpdate),
						},
					},
					ConfigStateChecks: []statecheck.StateCheck{
						statecheck.ExpectKnownValue(bbAddr.String(), tfjsonpath.New("spec").AtMapKey("inputs").AtMapKey("environment").AtMapKey("value"), knownvalue.StringExact("staging")),
					},
				},
				{
					Config: retriggerConfig.String(),
					ConfigPlanChecks: resource.ConfigPlanChecks{
						PreApply: []plancheck.PlanCheck{
							plancheck.ExpectResourceAction(bbAddr.String(), plancheck.ResourceActionUpdate),
						},
					},
					ConfigStateChecks: []statecheck.StateCheck{
						statecheck.ExpectKnownValue(bbAddr.String(), tfjsonpath.New("spec").AtMapKey("inputs").AtMapKey("environment").AtMapKey("value"), knownvalue.StringExact("staging")),
						statecheck.ExpectKnownValue(bbAddr.String(), tfjsonpath.New("retrigger_run"), knownvalue.StringExact("force-run-1")),
					},
				},
			},
		})
	})

	t.Run("02_tenant", func(t *testing.T) {
		config, bbAddr := testconfig.BuildBBv3TenantConfig(t)

		ApplyAndTest(t, resource.TestCase{
			Steps: []resource.TestStep{
				{
					Config: config.String(),
					ConfigPlanChecks: resource.ConfigPlanChecks{
						PreApply: []plancheck.PlanCheck{
							plancheck.ExpectResourceAction(bbAddr.String(), plancheck.ResourceActionCreate),
						},
					},
					ConfigStateChecks: bbv3StateChecks(bbAddr, "my-tenant-building-block"),
				},
				{
					ImportState:     true,
					ImportStateKind: resource.ImportBlockWithID,
					ImportStateIdFunc: func(s *terraform.State) (string, error) {
						rs := s.RootModule().Resources[bbAddr.String()]
						if rs == nil {
							return "", fmt.Errorf("resource not found: %s", bbAddr.String())
						}
						return rs.Primary.Attributes["metadata.uuid"], nil
					},
					ResourceName: bbAddr.String(),
				},
			},
		})
	})

	t.Run("03_workspace_moved_from_v2", func(t *testing.T) {
		workspaceConfig, workspaceAddr := testconfig.BuildWorkspaceConfig(t)

		var bbdAddr testconfig.Traversal
		bbdConfig := testconfig.Resource{Name: "building_block_v2", Suffix: "_01_workspace"}.TestSupportConfig(t, "").WithFirstBlock(t,
			testconfig.ExtractIdentifier(&bbdAddr),
			testconfig.OwnedByWorkspace(t, workspaceAddr),
		)

		var v2Addr testconfig.Traversal
		v2Config := testconfig.Resource{Name: "building_block_v2", Suffix: "_01_workspace"}.Config(t).WithFirstBlock(t,
			testconfig.ExtractIdentifier(&v2Addr),
			testconfig.Traverse(t, "spec", "building_block_definition_version_ref")(testconfig.SetRawExpr(bbdAddr.Join("version_latest").String())),
			testconfig.Traverse(t, "spec", "target_ref")(testconfig.SetRawExpr(workspaceAddr.Join("ref").String())),
		).Join(workspaceConfig, bbdConfig)

		var v3Addr testconfig.Traversal
		v3Config := testconfig.Resource{Name: "building_block_v3", Suffix: "_01_workspace"}.Config(t).WithFirstBlock(t,
			testconfig.ExtractIdentifier(&v3Addr),
			testconfig.Traverse(t, "spec", "building_block_definition_version_ref")(testconfig.SetRawExpr(bbdAddr.Join("version_latest").String())),
			testconfig.Traverse(t, "spec", "target_ref")(testconfig.SetRawExpr(workspaceAddr.Join("ref").String())),
		).Join(workspaceConfig, bbdConfig)

		movedConfig := testconfig.NewConfig(t, []byte(fmt.Sprintf(`
moved {
  from = %s
  to   = %s
}
`, v2Addr.String(), v3Addr.String())))

		ApplyAndTest(t, resource.TestCase{
			Steps: []resource.TestStep{
				{
					Config: v2Config.String(),
					ConfigPlanChecks: resource.ConfigPlanChecks{
						PreApply: []plancheck.PlanCheck{
							plancheck.ExpectResourceAction(v2Addr.String(), plancheck.ResourceActionCreate),
						},
					},
					ConfigStateChecks: []statecheck.StateCheck{
						statecheck.ExpectKnownValue(v2Addr.String(), tfjsonpath.New("metadata").AtMapKey("uuid"), xknownvalue.NotEmptyString()),
					},
				},
				{
					Config:            v3Config.Join(movedConfig).String(),
					ConfigStateChecks: bbv3StateChecks(v3Addr, "my-workspace-building-block"),
				},
			},
		})
	})

	t.Run("04_tenant_moved_from_v1", func(t *testing.T) {
		if !IsMockClientTest() {
			t.Skip("Skipping: BB v1 resource has no wait_for_completion, BB run stays PENDING and blocks destroy")
		}

		workspaceConfig, workspaceAddr := testconfig.BuildWorkspaceConfig(t)
		projectConfig, projectAddr := testconfig.BuildProjectConfig(t, workspaceAddr)
		platformConfig, platformAddr, platformTypeAddr := testconfig.BuildCustomPlatformConfig(t, workspaceAddr)
		landingZoneConfig, landingZoneAddr := testconfig.BuildLandingZoneConfig(t, workspaceAddr, platformAddr, platformTypeAddr)

		var tenantAddr testconfig.Traversal
		tenantConfig := testconfig.Resource{Name: "tenant_v4"}.Config(t).WithFirstBlock(t,
			testconfig.ExtractIdentifier(&tenantAddr),
			testconfig.Traverse(t, "metadata")(
				testconfig.Traverse(t, "owned_by_workspace")(testconfig.SetRawExpr(projectAddr.Join("metadata", "owned_by_workspace").String())),
				testconfig.Traverse(t, "owned_by_project")(testconfig.SetRawExpr(projectAddr.Join("metadata", "name").String())),
			),
			testconfig.Traverse(t, "spec")(
				testconfig.Traverse(t, "platform_identifier")(testconfig.SetRawExpr(platformAddr.Join("identifier").String())),
				testconfig.Traverse(t, "landing_zone_identifier")(testconfig.SetRawExpr(landingZoneAddr.Join("metadata", "name").String())),
			),
		)

		var bbdAddr testconfig.Traversal
		bbdConfig := testconfig.Resource{Name: "building_block_v3", Suffix: "_02_tenant"}.TestSupportConfig(t, "").WithFirstBlock(t,
			testconfig.ExtractIdentifier(&bbdAddr),
			testconfig.OwnedByWorkspace(t, workspaceAddr),
			testconfig.Traverse(t, "spec", "supported_platforms")(testconfig.SetRawExpr(fmt.Sprintf("[{name = %s}]", platformTypeAddr.Join("metadata", "name")))),
		)

		var v1Addr testconfig.Traversal
		v1Config := testconfig.Resource{Name: "buildingblock"}.Config(t).WithFirstBlock(t,
			testconfig.ExtractIdentifier(&v1Addr),
			testconfig.Traverse(t, "metadata")(
				testconfig.Traverse(t, "definition_uuid")(testconfig.SetRawExpr(bbdAddr.Join("ref", "uuid").String())),
				testconfig.Traverse(t, "definition_version")(testconfig.SetRawExpr(bbdAddr.Join("version_latest", "number").String())),
				testconfig.Traverse(t, "tenant_identifier")(testconfig.SetRawExpr(tenantAddr.Format(
					`"${%s.metadata.owned_by_workspace}.${%s.metadata.owned_by_project}.${%s.spec.platform_identifier}"`,
					tenantAddr.String(), tenantAddr.String(),
				))),
			),
			testconfig.Traverse(t, "spec", "inputs", "environment")(testconfig.SetRawExpr(`{ value_single_select = "dev" }`)),
		).Join(workspaceConfig, projectConfig, platformConfig, landingZoneConfig, tenantConfig, bbdConfig)

		var v3Addr testconfig.Traversal
		v3Config := testconfig.Resource{Name: "building_block_v3", Suffix: "_02_tenant"}.Config(t).WithFirstBlock(t,
			testconfig.ExtractIdentifier(&v3Addr),
			testconfig.Traverse(t, "spec", "building_block_definition_version_ref")(testconfig.SetRawExpr(bbdAddr.Join("version_latest").String())),
			testconfig.Traverse(t, "spec", "target_ref")(testconfig.SetRawExpr(tenantAddr.Join("ref").String())),
		).Join(workspaceConfig, projectConfig, platformConfig, landingZoneConfig, tenantConfig, bbdConfig)

		movedConfig := testconfig.NewConfig(t, []byte(fmt.Sprintf(`
moved {
  from = %s
  to   = %s
}
`, v1Addr.String(), v3Addr.String())))

		ApplyAndTest(t, resource.TestCase{
			Steps: []resource.TestStep{
				{
					Config: v1Config.String(),
					ConfigPlanChecks: resource.ConfigPlanChecks{
						PreApply: []plancheck.PlanCheck{
							plancheck.ExpectResourceAction(v1Addr.String(), plancheck.ResourceActionCreate),
						},
					},
					ConfigStateChecks: []statecheck.StateCheck{
						statecheck.ExpectKnownValue(v1Addr.String(), tfjsonpath.New("metadata").AtMapKey("uuid"), xknownvalue.NotEmptyString()),
					},
				},
				{
					Config:            v3Config.Join(movedConfig).String(),
					ConfigStateChecks: bbv3StateChecks(v3Addr, "my-tenant-building-block"),
				},
			},
		})
	})
}

func bbv3StateChecks(bbAddr testconfig.Traversal, displayName string) []statecheck.StateCheck {
	return []statecheck.StateCheck{
		statecheck.ExpectKnownValue(bbAddr.String(), tfjsonpath.New("metadata").AtMapKey("uuid"), xknownvalue.NotEmptyString()),
		statecheck.ExpectKnownValue(bbAddr.String(), tfjsonpath.New("spec").AtMapKey("display_name"), knownvalue.StringExact(displayName)),
		statecheck.ExpectKnownValue(bbAddr.String(), tfjsonpath.New("spec").AtMapKey("inputs").AtMapKey("name").AtMapKey("value"), knownvalue.StringExact("my-name")),
		statecheck.ExpectKnownValue(bbAddr.String(), tfjsonpath.New("spec").AtMapKey("inputs").AtMapKey("size").AtMapKey("value"), knownvalue.StringExact("16")),
		statecheck.ExpectKnownValue(bbAddr.String(), tfjsonpath.New("spec").AtMapKey("inputs").AtMapKey("environment").AtMapKey("value"), knownvalue.StringExact("dev")),
		statecheck.ExpectKnownValue(bbAddr.String(), tfjsonpath.New("status").AtMapKey("status"), knownvalue.StringExact("SUCCEEDED")),
	}
}
