package provider

import (
	"fmt"
	"regexp"
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
		config, buildingBlockAddr := testconfig.BBv3Workspace(t)
		updatedInputsConfig := config.WithFirstBlock(
			testconfig.Descend("spec", "inputs", "environment", "value")(testconfig.SetString("staging")),
		)
		retriggerConfig := updatedInputsConfig.WithFirstBlock(
			testconfig.Descend("retrigger_run")(testconfig.SetString("force-run-1")),
		)

		ApplyAndTest(t, resource.TestCase{
			Steps: []resource.TestStep{
				{
					Config: config.String(),
					ConfigPlanChecks: resource.ConfigPlanChecks{
						PreApply: []plancheck.PlanCheck{
							plancheck.ExpectResourceAction(buildingBlockAddr.String(), plancheck.ResourceActionCreate),
						},
					},
					ConfigStateChecks: bbv3StateChecks(buildingBlockAddr, "my-workspace-building-block"),
				},
				{
					ImportState:     true,
					ImportStateKind: resource.ImportBlockWithID,
					ImportStateIdFunc: func(s *terraform.State) (string, error) {
						rs := s.RootModule().Resources[buildingBlockAddr.String()]
						if rs == nil {
							return "", fmt.Errorf("resource not found: %s", buildingBlockAddr.String())
						}
						return rs.Primary.Attributes["metadata.uuid"], nil
					},
					ResourceName: buildingBlockAddr.String(),
				},
				{
					Config: updatedInputsConfig.String(),
					ConfigPlanChecks: resource.ConfigPlanChecks{
						PreApply: []plancheck.PlanCheck{
							plancheck.ExpectResourceAction(buildingBlockAddr.String(), plancheck.ResourceActionUpdate),
						},
					},
					ConfigStateChecks: []statecheck.StateCheck{
						statecheck.ExpectKnownValue(buildingBlockAddr.String(), tfjsonpath.New("spec").AtMapKey("inputs").AtMapKey("environment").AtMapKey("value"), knownvalue.StringExact("staging")),
					},
				},
				{
					Config: retriggerConfig.String(),
					ConfigPlanChecks: resource.ConfigPlanChecks{
						PreApply: []plancheck.PlanCheck{
							plancheck.ExpectResourceAction(buildingBlockAddr.String(), plancheck.ResourceActionUpdate),
						},
					},
					ConfigStateChecks: []statecheck.StateCheck{
						statecheck.ExpectKnownValue(buildingBlockAddr.String(), tfjsonpath.New("spec").AtMapKey("inputs").AtMapKey("environment").AtMapKey("value"), knownvalue.StringExact("staging")),
						statecheck.ExpectKnownValue(buildingBlockAddr.String(), tfjsonpath.New("retrigger_run"), knownvalue.StringExact("force-run-1")),
					},
				},
			},
		})
	})

	t.Run("02_tenant", func(t *testing.T) {
		config, buildingBlockAddr := testconfig.BBv3Tenant(t)

		ApplyAndTest(t, resource.TestCase{
			Steps: []resource.TestStep{
				{
					Config: config.String(),
					ConfigPlanChecks: resource.ConfigPlanChecks{
						PreApply: []plancheck.PlanCheck{
							plancheck.ExpectResourceAction(buildingBlockAddr.String(), plancheck.ResourceActionCreate),
						},
					},
					ConfigStateChecks: bbv3StateChecks(buildingBlockAddr, "my-tenant-building-block"),
				},
				{
					ImportState:     true,
					ImportStateKind: resource.ImportBlockWithID,
					ImportStateIdFunc: func(s *terraform.State) (string, error) {
						rs := s.RootModule().Resources[buildingBlockAddr.String()]
						if rs == nil {
							return "", fmt.Errorf("resource not found: %s", buildingBlockAddr.String())
						}
						return rs.Primary.Attributes["metadata.uuid"], nil
					},
					ResourceName: buildingBlockAddr.String(),
				},
			},
		})
	})

	t.Run("03_workspace_moved_from_v2", func(t *testing.T) {
		workspaceConfig, workspaceAddr := testconfig.Workspace(t)

		var buildingBlockDefinitionAddr testconfig.Traversal
		buildingBlockDefinitionConfig := testconfig.Resource{Name: "building_block_v2", Suffix: "_01_workspace"}.TestSupportConfig(t, "").WithFirstBlock(
			testconfig.ExtractAddress(&buildingBlockDefinitionAddr),
			testconfig.OwnedByWorkspace(workspaceAddr),
		)

		var v2Addr testconfig.Traversal
		v2Config := testconfig.Resource{Name: "building_block_v2", Suffix: "_01_workspace"}.Config(t).WithFirstBlock(
			testconfig.ExtractAddress(&v2Addr),
			testconfig.Descend("spec", "building_block_definition_version_ref")(testconfig.SetAddr(buildingBlockDefinitionAddr, "version_latest")),
			testconfig.Descend("spec", "target_ref")(testconfig.SetAddr(workspaceAddr, "ref")),
		).Join(workspaceConfig, buildingBlockDefinitionConfig)

		var v3Addr testconfig.Traversal
		v3Config := testconfig.Resource{Name: "building_block_v3", Suffix: "_01_workspace"}.Config(t).WithFirstBlock(
			testconfig.ExtractAddress(&v3Addr),
			testconfig.Descend("spec", "building_block_definition_version_ref")(testconfig.SetAddr(buildingBlockDefinitionAddr, "version_latest")),
			testconfig.Descend("spec", "target_ref")(testconfig.SetAddr(workspaceAddr, "ref")),
		).Join(workspaceConfig, buildingBlockDefinitionConfig)

		movedConfig := testconfig.Resource{Name: "building_block_v3"}.TestSupportConfig(t, "_moved_from_v2").WithFirstBlock(
			testconfig.Descend("from")(testconfig.SetAddr(v2Addr)),
			testconfig.Descend("to")(testconfig.SetAddr(v3Addr)),
		)

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

		workspaceConfig, workspaceAddr := testconfig.Workspace(t)
		projectConfig, projectAddr := testconfig.Project(t, workspaceAddr)
		platformConfig, platformAddr, platformTypeAddr := testconfig.CustomPlatform(t, workspaceAddr)
		landingZoneConfig, landingZoneAddr := testconfig.LandingZone(t, workspaceAddr, platformAddr, platformTypeAddr)

		var tenantAddr testconfig.Traversal
		tenantConfig := testconfig.Resource{Name: "tenant_v4"}.Config(t).WithFirstBlock(
			testconfig.ExtractAddress(&tenantAddr),
			testconfig.Descend("metadata")(
				testconfig.Descend("owned_by_workspace")(testconfig.SetAddr(projectAddr, "metadata", "owned_by_workspace")),
				testconfig.Descend("owned_by_project")(testconfig.SetAddr(projectAddr, "metadata", "name")),
			),
			testconfig.Descend("spec")(
				testconfig.Descend("platform_identifier")(testconfig.SetAddr(platformAddr, "identifier")),
				testconfig.Descend("landing_zone_identifier")(testconfig.SetAddr(landingZoneAddr, "metadata", "name")),
			),
		)

		var buildingBlockDefinitionAddr testconfig.Traversal
		buildingBlockDefinitionConfig := testconfig.Resource{Name: "building_block_v3", Suffix: "_02_tenant"}.TestSupportConfig(t, "").WithFirstBlock(
			testconfig.ExtractAddress(&buildingBlockDefinitionAddr),
			testconfig.OwnedByWorkspace(workspaceAddr),
			testconfig.Descend("spec", "supported_platforms")(testconfig.SetRawExpr("[{name = %s}]", platformTypeAddr.Join("metadata", "name"))),
		)

		var v1Addr testconfig.Traversal
		v1Config := testconfig.Resource{Name: "buildingblock"}.Config(t).WithFirstBlock(
			testconfig.ExtractAddress(&v1Addr),
			testconfig.Descend("metadata")(
				testconfig.Descend("definition_uuid")(testconfig.SetAddr(buildingBlockDefinitionAddr, "ref", "uuid")),
				testconfig.Descend("definition_version")(testconfig.SetAddr(buildingBlockDefinitionAddr, "version_latest", "number")),
				testconfig.Descend("tenant_identifier")(testconfig.SetRawExpr(
					`"${%s.metadata.owned_by_workspace}.${%s.metadata.owned_by_project}.${%s.spec.platform_identifier}"`,
					tenantAddr, tenantAddr, tenantAddr,
				)),
			),
			testconfig.Descend("spec", "inputs", "environment")(testconfig.SetRawExpr(`{ value_single_select = "dev" }`)),
		).Join(workspaceConfig, projectConfig, platformConfig, landingZoneConfig, tenantConfig, buildingBlockDefinitionConfig)

		var v3Addr testconfig.Traversal
		v3Config := testconfig.Resource{Name: "building_block_v3", Suffix: "_02_tenant"}.Config(t).WithFirstBlock(
			testconfig.ExtractAddress(&v3Addr),
			testconfig.Descend("spec", "building_block_definition_version_ref")(testconfig.SetAddr(buildingBlockDefinitionAddr, "version_latest")),
			testconfig.Descend("spec", "target_ref")(testconfig.SetAddr(tenantAddr, "ref")),
		).Join(workspaceConfig, projectConfig, platformConfig, landingZoneConfig, tenantConfig, buildingBlockDefinitionConfig)

		movedConfig := testconfig.Resource{Name: "building_block_v3"}.TestSupportConfig(t, "_moved_from_v1").WithFirstBlock(
			testconfig.Descend("from")(testconfig.SetAddr(v1Addr)),
			testconfig.Descend("to")(testconfig.SetAddr(v3Addr)),
		)

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

	t.Run("05_workspace_invalid_input_assignment", func(t *testing.T) {
		config, _ := testconfig.BBv3Workspace(t)
		invalidConfig := config.WithFirstBlock(
			testconfig.Descend("spec", "inputs", "region")(testconfig.SetRawExpr(`{
  value = "eu-central-1"
}`)),
		)

		ApplyAndTest(t, resource.TestCase{
			Steps: []resource.TestStep{
				{
					Config:      invalidConfig.String(),
					ExpectError: regexp.MustCompile("Input configured in wrong attribute"),
				},
			},
		})
	})
}

func bbv3StateChecks(buildingBlockAddr testconfig.Traversal, displayName string) []statecheck.StateCheck {
	checks := []statecheck.StateCheck{
		statecheck.ExpectKnownValue(buildingBlockAddr.String(), tfjsonpath.New("metadata").AtMapKey("uuid"), xknownvalue.NotEmptyString()),
		statecheck.ExpectKnownValue(buildingBlockAddr.String(), tfjsonpath.New("spec").AtMapKey("display_name"), knownvalue.StringExact(displayName)),
		statecheck.ExpectKnownValue(buildingBlockAddr.String(), tfjsonpath.New("spec").AtMapKey("inputs").AtMapKey("name").AtMapKey("value"), knownvalue.StringExact("my-name")),
		statecheck.ExpectKnownValue(buildingBlockAddr.String(), tfjsonpath.New("spec").AtMapKey("inputs").AtMapKey("size").AtMapKey("value"), knownvalue.StringExact("16")),
		statecheck.ExpectKnownValue(buildingBlockAddr.String(), tfjsonpath.New("spec").AtMapKey("inputs").AtMapKey("environment").AtMapKey("value"), knownvalue.StringExact("dev")),
		statecheck.ExpectKnownValue(buildingBlockAddr.String(), tfjsonpath.New("status").AtMapKey("status"), xknownvalue.NotEmptyString()),
	}
	if IsMockClientTest() {
		checks = append(checks, statecheck.ExpectKnownValue(buildingBlockAddr.String(), tfjsonpath.New("status").AtMapKey("latest_run").AtMapKey("uuid"), xknownvalue.NotEmptyString()))
	}
	return checks
}
