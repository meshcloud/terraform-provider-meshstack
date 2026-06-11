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
	RequireDevMeshStack(t)
	t.Parallel()

	t.Run("01_workspace", func(t *testing.T) {
		config, buildingBlockAddr := testconfig.BBv2Workspace(t)

		ApplyAndTest(t, resource.TestCase{
			Steps: []resource.TestStep{
				{
					Config: config.String(),
					ConfigPlanChecks: resource.ConfigPlanChecks{
						PreApply: []plancheck.PlanCheck{
							plancheck.ExpectResourceAction(buildingBlockAddr.String(), plancheck.ResourceActionCreate),
						},
					},
					ConfigStateChecks: bbv2StateChecks(buildingBlockAddr, "my-workspace-building-block"),
				},
			},
		})
	})

	t.Run("02_tenant", func(t *testing.T) {
		config, buildingBlockAddr := testconfig.BBv2Tenant(t)

		ApplyAndTest(t, resource.TestCase{
			Steps: []resource.TestStep{
				{
					Config: config.String(),
					ConfigPlanChecks: resource.ConfigPlanChecks{
						PreApply: []plancheck.PlanCheck{
							plancheck.ExpectResourceAction(buildingBlockAddr.String(), plancheck.ResourceActionCreate),
						},
					},
					ConfigStateChecks: bbv2StateChecks(buildingBlockAddr, "my-tenant-building-block"),
				},
			},
		})
	})
	t.Run("03_sensitive_input", func(t *testing.T) {
		bbv2SensitiveInputSubtest(t)
	})

	t.Run("04_sensitive_user_input", func(t *testing.T) {
		bbv2SensitiveUserInputSubtest(t)
	})
}

func TestAccBuildingBlockV2Local(t *testing.T) {
	RequireLocalMeshStack(t)
	t.Parallel()

	t.Run("01_workspace", func(t *testing.T) {
		config, buildingBlockAddr := testconfig.BBv2WorkspaceLocal(t)

		ApplyAndTest(t, resource.TestCase{
			Steps: []resource.TestStep{
				{
					Config: config.String(),
					ConfigPlanChecks: resource.ConfigPlanChecks{
						PreApply: []plancheck.PlanCheck{
							plancheck.ExpectResourceAction(buildingBlockAddr.String(), plancheck.ResourceActionCreate),
						},
					},
					ConfigStateChecks: bbv2StateChecks(buildingBlockAddr, "my-workspace-building-block"),
				},
			},
		})
	})

	t.Run("02_tenant", func(t *testing.T) {
		config, buildingBlockAddr := testconfig.BBv2TenantLocal(t)

		ApplyAndTest(t, resource.TestCase{
			Steps: []resource.TestStep{
				{
					Config: config.String(),
					ConfigPlanChecks: resource.ConfigPlanChecks{
						PreApply: []plancheck.PlanCheck{
							plancheck.ExpectResourceAction(buildingBlockAddr.String(), plancheck.ResourceActionCreate),
						},
					},
					ConfigStateChecks: bbv2StateChecks(buildingBlockAddr, "my-tenant-building-block"),
				},
			},
		})
	})

	runnerMod := testconfig.BBDRunnerRef(LocalBuildingBlockRunnerUuid)

	t.Run("03_sensitive_input", func(t *testing.T) {
		bbv2SensitiveInputSubtest(t, runnerMod)
	})

	t.Run("04_sensitive_user_input", func(t *testing.T) {
		bbv2SensitiveUserInputSubtest(t, runnerMod)
	})
}

func bbv2SensitiveInputSubtest(t *testing.T, extraBBDMods ...testconfig.ExpressionConsumer) {
	t.Helper()
	if IsMockClientTest() {
		// The in-memory mock does not resolve STATIC inputs from the BBD, so the
		// static secret never appears in combined_inputs in mock mode.
		t.Skip("requires real meshStack to resolve static secret inputs")
	}

	workspaceConfig, workspaceAddr := testconfig.Workspace(t)
	exampleResource := testconfig.Resource{Name: "building_block_v2", Suffix: "_03_sensitive_input"}

	var buildingBlockDefinitionAddr testconfig.Traversal
	buildingBlockDefinitionConfig := exampleResource.TestSupportConfig(t, "_bbd").WithFirstBlock(
		append([]testconfig.ExpressionConsumer{
			testconfig.ExtractAddress(&buildingBlockDefinitionAddr),
			testconfig.OwnedByWorkspace(workspaceAddr),
		}, extraBBDMods...)...,
	)

	var buildingBlockAddr testconfig.Traversal
	config := exampleResource.TestSupportConfig(t, "").WithFirstBlock(
		testconfig.ExtractAddress(&buildingBlockAddr),
		testconfig.Descend("spec", "building_block_definition_version_ref")(testconfig.SetAddr(buildingBlockDefinitionAddr, "version_latest")),
		testconfig.Descend("spec", "target_ref")(testconfig.SetAddr(workspaceAddr, "ref")),
	).Join(workspaceConfig, buildingBlockDefinitionConfig)

	ApplyAndTest(t, resource.TestCase{
		Steps: []resource.TestStep{
			{
				Config: config.String(),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction(buildingBlockAddr.String(), plancheck.ResourceActionCreate),
					},
				},
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(buildingBlockAddr.String(), tfjsonpath.New("metadata").AtMapKey("uuid"), xknownvalue.NotEmptyString()),
					// The read fix surfaces the embedded-secret hash here; without it this is null.
					statecheck.ExpectKnownValue(buildingBlockAddr.String(),
						tfjsonpath.New("spec").AtMapKey("combined_inputs").AtMapKey("static_secret").AtMapKey("value_string"),
						xknownvalue.NotEmptyString()),
				},
			},
		},
	})
}

func bbv2SensitiveUserInputSubtest(t *testing.T, extraBBDMods ...testconfig.ExpressionConsumer) {
	t.Helper()
	if IsMockClientTest() {
		// The in-memory mock does not process SecretEmbedded plaintext, so the hash
		// never appears in combined_inputs in mock mode.
		t.Skip("requires real meshStack to process sensitive user inputs")
	}

	workspaceConfig, workspaceAddr := testconfig.Workspace(t)
	exampleResource := testconfig.Resource{Name: "building_block_v2", Suffix: "_04_sensitive_user_input"}

	var buildingBlockDefinitionAddr testconfig.Traversal
	buildingBlockDefinitionConfig := exampleResource.TestSupportConfig(t, "_bbd").WithFirstBlock(
		append([]testconfig.ExpressionConsumer{
			testconfig.ExtractAddress(&buildingBlockDefinitionAddr),
			testconfig.OwnedByWorkspace(workspaceAddr),
		}, extraBBDMods...)...,
	)

	var buildingBlockAddr testconfig.Traversal
	config := exampleResource.TestSupportConfig(t, "").WithFirstBlock(
		testconfig.ExtractAddress(&buildingBlockAddr),
		testconfig.Descend("spec", "building_block_definition_version_ref")(testconfig.SetAddr(buildingBlockDefinitionAddr, "version_latest")),
		testconfig.Descend("spec", "target_ref")(testconfig.SetAddr(workspaceAddr, "ref")),
	).Join(workspaceConfig, buildingBlockDefinitionConfig)

	ApplyAndTest(t, resource.TestCase{
		Steps: []resource.TestStep{
			{
				Config: config.String(),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction(buildingBlockAddr.String(), plancheck.ResourceActionCreate),
					},
				},
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(buildingBlockAddr.String(), tfjsonpath.New("metadata").AtMapKey("uuid"), xknownvalue.NotEmptyString()),
					// Sensitive user inputs are sent as {"plaintext":...}; the API returns the hash.
					// The hash surfaces in combined_inputs (the STRING hash in value_string, the CODE hash in value_code).
					statecheck.ExpectKnownValue(buildingBlockAddr.String(),
						tfjsonpath.New("spec").AtMapKey("combined_inputs").AtMapKey("secret_str").AtMapKey("value_string"),
						xknownvalue.NotEmptyString()),
					statecheck.ExpectKnownValue(buildingBlockAddr.String(),
						tfjsonpath.New("spec").AtMapKey("combined_inputs").AtMapKey("secret_code").AtMapKey("value_code"),
						xknownvalue.NotEmptyString()),
				},
			},
		},
	})
}

func bbv2StateChecks(buildingBlockAddr testconfig.Traversal, displayName string) []statecheck.StateCheck {
	return []statecheck.StateCheck{
		statecheck.ExpectKnownValue(buildingBlockAddr.String(), tfjsonpath.New("metadata").AtMapKey("uuid"), xknownvalue.NotEmptyString()),
		statecheck.ExpectKnownValue(buildingBlockAddr.String(), tfjsonpath.New("spec").AtMapKey("display_name"), knownvalue.StringExact(displayName)),
		statecheck.ExpectKnownValue(buildingBlockAddr.String(), tfjsonpath.New("spec").AtMapKey("inputs").AtMapKey("name").AtMapKey("value_string"), knownvalue.StringExact("my-name")),
		statecheck.ExpectKnownValue(buildingBlockAddr.String(), tfjsonpath.New("spec").AtMapKey("inputs").AtMapKey("size").AtMapKey("value_int"), knownvalue.Int64Exact(16)),
		statecheck.ExpectKnownValue(buildingBlockAddr.String(), tfjsonpath.New("spec").AtMapKey("inputs").AtMapKey("environment").AtMapKey("value_single_select"), knownvalue.StringExact("dev")),
		statecheck.ExpectKnownValue(buildingBlockAddr.String(), tfjsonpath.New("status").AtMapKey("status"), knownvalue.StringExact("SUCCEEDED")),
	}
}
