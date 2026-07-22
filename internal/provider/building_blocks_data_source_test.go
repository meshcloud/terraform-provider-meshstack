package provider

import (
	"fmt"
	"testing"

	tfconfig "github.com/hashicorp/terraform-plugin-testing/config"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/knownvalue"
	"github.com/hashicorp/terraform-plugin-testing/statecheck"
	"github.com/hashicorp/terraform-plugin-testing/tfjsonpath"

	"github.com/meshcloud/terraform-provider-meshstack/internal/provider/acctest/testconfig"
	"github.com/meshcloud/terraform-provider-meshstack/internal/provider/acctest/xknownvalue"
)

// TestAccBuildingBlocksDataSource creates a real building block with the BBWorkspace builder and
// lists it back through the data source. Like every other data source test in this package, it
// reuses a testconfig builder to actually create the resources under test (rather than
// pre-populating a mock), so it runs identically in mock mode and as a true acceptance test
// (TF_ACC=1) against a local meshStack. Filtering by the freshly-created workspace yields exactly
// the one block, so indexing building_blocks.0 is deterministic; referencing buildingBlockAddr in
// the filter makes Terraform read the data source only after the block exists.
func TestAccBuildingBlocksDataSource(t *testing.T) {
	t.Parallel()

	t.Run("01_workspace", func(t *testing.T) {
		buildingBlockConfig, buildingBlockAddr, _, _ := testconfig.BBWorkspace(t)

		dataSourceAddr := "data.meshstack_building_blocks.all"
		config := testconfig.DataSource{Name: "building_blocks"}.Config(t).WithFirstBlock(
			testconfig.Descend("workspace_identifier")(testconfig.SetAddr(buildingBlockAddr, "metadata", "owned_by_workspace")),
		).Join(buildingBlockConfig)

		ApplyAndTest(t, resource.TestCase{
			Steps: []resource.TestStep{
				{
					Config: config.String(),
					ConfigStateChecks: []statecheck.StateCheck{
						statecheck.ExpectKnownValue(dataSourceAddr, tfjsonpath.New("building_blocks"), knownvalue.ListSizeExact(1)),
						statecheck.ExpectKnownValue(dataSourceAddr, tfjsonpath.New("building_blocks").AtSliceIndex(0).AtMapKey("metadata").AtMapKey("uuid"), xknownvalue.NotEmptyString()),
						statecheck.ExpectKnownValue(dataSourceAddr, tfjsonpath.New("building_blocks").AtSliceIndex(0).AtMapKey("spec").AtMapKey("display_name"), knownvalue.StringExact("my-workspace-building-block")),
						// all_inputs surfaces every backend input read-only (the _01_workspace BBD declares size + environment).
						statecheck.ExpectKnownValue(dataSourceAddr, tfjsonpath.New("building_blocks").AtSliceIndex(0).AtMapKey("all_inputs").AtMapKey("size").AtMapKey("value"), knownvalue.StringExact("16")),
						statecheck.ExpectKnownValue(dataSourceAddr, tfjsonpath.New("building_blocks").AtSliceIndex(0).AtMapKey("all_inputs").AtMapKey("environment").AtMapKey("value"), knownvalue.StringExact(`"dev"`)),
					},
				},
			},
		})
	})

	// 02_version_number_filter exercises the server-side version_number filter (the meshfed change).
	// The block is created from version 1 of its definition, so filtering by the lenient "v1" must
	// return it while "v2" must return nothing. This is acceptance-only: the filter is applied by the
	// backend (the mock store carries only the definition *version uuid*, not the version number), and
	// the "v2 → empty" assertion specifically proves the backend parses + applies the new param — an
	// older backend without it would ignore versionNumber and return the block for both values.
	t.Run("02_version_number_filter", func(t *testing.T) {
		if IsMockClientTest() {
			t.Skip("version_number is filtered server-side; the mock store does not carry the BBD version number")
		}
		buildingBlockConfig, buildingBlockAddr, _, _ := testconfig.BBWorkspace(t)
		dataSourceAddr := "data.meshstack_building_blocks.all"

		base := func(versionNumber string) testconfig.Config {
			return testconfig.DataSource{Name: "building_blocks"}.Config(t).WithFirstBlock(
				testconfig.Descend("workspace_identifier")(testconfig.SetAddr(buildingBlockAddr, "metadata", "owned_by_workspace")),
				testconfig.Descend("version_number")(testconfig.SetRawExpr(`%q`, versionNumber)),
			).Join(buildingBlockConfig)
		}

		ApplyAndTest(t, resource.TestCase{
			Steps: []resource.TestStep{
				{
					// Lenient "v1" matches definition version 1.
					Config: base("v1").String(),
					ConfigStateChecks: []statecheck.StateCheck{
						statecheck.ExpectKnownValue(dataSourceAddr, tfjsonpath.New("building_blocks"), knownvalue.ListSizeExact(1)),
						statecheck.ExpectKnownValue(dataSourceAddr, tfjsonpath.New("building_blocks").AtSliceIndex(0).AtMapKey("spec").AtMapKey("display_name"), knownvalue.StringExact("my-workspace-building-block")),
					},
				},
				{
					// Version 2 does not exist for this block → empty result (proves the param is applied).
					Config: base("v2").String(),
					ConfigStateChecks: []statecheck.StateCheck{
						statecheck.ExpectKnownValue(dataSourceAddr, tfjsonpath.New("building_blocks"), knownvalue.ListSizeExact(0)),
					},
				},
			},
		})
	})

	// 03_permission_scoped_list exercises the meshfed change that scopes the *unfiltered* meshBuildingBlock
	// v2 list by the caller's authority (one-permission-one-scope): MANAGED_BUILDINGBLOCK_LIST alone returns
	// the blocks the caller manages (created from a definition its workspace owns) even across the workspace
	// boundary, while BUILDINGBLOCK_LIST alone returns only the caller's own/consumed blocks. Workspace A
	// owns the definition (manages); the block lives in the "other" workspace B (owned/consumed there); both
	// API keys belong to A. So A's managed scope contains the block and A's own scope does not — an older
	// backend, where an unfiltered managed-capable list only ever returned own/consumed blocks, would return
	// nothing for the managed key. Acceptance-only: the mock client enforces no authority boundaries.
	t.Run("03_permission_scoped_list", func(t *testing.T) {
		if IsMockClientTest() {
			t.Skip("permission-scoped listing requires real authority boundaries")
		}

		// Workspace A owns the BBD and therefore manages every block created from it.
		workspaceConfig, workspaceAddr := testconfig.Workspace(t)
		var bbdAddr testconfig.Traversal
		bbdConfig := testconfig.Resource{Name: "building_block", Suffix: "_01_workspace"}.TestSupportConfig(t, "").WithFirstBlock(
			testconfig.ExtractAddress(&bbdAddr),
			testconfig.OwnedByWorkspace(workspaceAddr),
		)

		// Workspace B consumes: the block lives here, across the boundary from the definition owner.
		var otherWorkspaceAddr testconfig.Traversal
		otherWorkspaceConfig, _ := testconfig.Workspace(t)
		otherWorkspaceConfig = otherWorkspaceConfig.WithFirstBlock(
			testconfig.RenameKey("other"),
			testconfig.ExtractAddress(&otherWorkspaceAddr),
		)

		// Two keys owned by A, each carrying exactly one list authority so the unfiltered list resolves to a
		// single scope.
		managedKeyConfig, managedKeyAddr := testconfig.ApiKey(t, workspaceAddr)
		managedKeyConfig = managedKeyConfig.WithFirstBlock(
			testconfig.RenameKey("managed"),
			testconfig.ExtractAddress(&managedKeyAddr),
			testconfig.Descend("spec", "permissions")(testconfig.SetRawExpr(`["MANAGED_BUILDINGBLOCK_LIST"]`)),
		)
		ownKeyConfig, ownKeyAddr := testconfig.ApiKey(t, workspaceAddr)
		ownKeyConfig = ownKeyConfig.WithFirstBlock(
			testconfig.RenameKey("own"),
			testconfig.ExtractAddress(&ownKeyAddr),
			testconfig.Descend("spec", "permissions")(testconfig.SetRawExpr(`["BUILDINGBLOCK_LIST"]`)),
		)

		// Admin-managed block in workspace B from A's definition (created/destroyed by the default provider's
		// ADM_BUILDINGBLOCK_SAVE; the minted keys are only ever used to read the list).
		var bbAddr testconfig.Traversal
		bbConfig := testconfig.Resource{Name: "building_block", Suffix: "_01_workspace"}.Config(t).WithFirstBlock(
			testconfig.ExtractAddress(&bbAddr),
			testconfig.Descend("spec", "building_block_definition_version_ref")(testconfig.SetRawExpr(`{ uuid = %s }`, bbdAddr.Join("version_latest", "uuid"))),
			testconfig.Descend("spec", "target_ref")(testconfig.SetAddr(otherWorkspaceAddr, "ref")),
		)

		support := workspaceConfig.Join(bbdConfig, otherWorkspaceConfig, managedKeyConfig, ownKeyConfig, bbConfig)
		otherProvider := testconfig.DataSource{Name: "building_blocks"}.TestSupportConfig(t, "_other_provider")

		// Fully unfiltered list read through the meshstack-other alias; the key credentials vary per step so
		// the same data source resolves under each authority in turn.
		listConfig := testconfig.DataSource{Name: "building_blocks"}.Config(t).WithFirstBlock(
			testconfig.Descend("provider")(testconfig.SetRawExpr("meshstack-other")),
		).Join(support, otherProvider)

		dataSourceAddr := "data.meshstack_building_blocks.all"

		var managedId, managedSecret, ownId, ownSecret lazyVariable
		var bbUuid string
		ApplyAndTest(t, resource.TestCase{Steps: []resource.TestStep{
			{
				// Admin: build the cross-workspace fixture and mint both keys.
				Config: support.String(),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(bbAddr.String(), tfjsonpath.New("metadata").AtMapKey("uuid"), xknownvalue.NotEmptyString(func(uuid string) error {
						bbUuid = uuid
						return nil
					})),
					statecheck.ExpectKnownValue(managedKeyAddr.String(), tfjsonpath.New("status").AtMapKey("client_id"), xknownvalue.NotEmptyString(func(v string) error {
						managedId = lazyVariable(v)
						return nil
					})),
					statecheck.ExpectKnownValue(managedKeyAddr.String(), tfjsonpath.New("status").AtMapKey("client_secret"), xknownvalue.NotEmptyString(func(v string) error {
						managedSecret = lazyVariable(v)
						return nil
					})),
					statecheck.ExpectKnownValue(ownKeyAddr.String(), tfjsonpath.New("status").AtMapKey("client_id"), xknownvalue.NotEmptyString(func(v string) error {
						ownId = lazyVariable(v)
						return nil
					})),
					statecheck.ExpectKnownValue(ownKeyAddr.String(), tfjsonpath.New("status").AtMapKey("client_secret"), xknownvalue.NotEmptyString(func(v string) error {
						ownSecret = lazyVariable(v)
						return nil
					})),
				},
			},
			{
				// MANAGED_BUILDINGBLOCK_LIST alone → the cross-workspace managed block is the one and only result.
				Config:          listConfig.String(),
				ConfigVariables: tfconfig.Variables{"apikey_client_id": &managedId, "apikey_client_secret": &managedSecret},
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(dataSourceAddr, tfjsonpath.New("building_blocks"), knownvalue.ListSizeExact(1)),
					statecheck.ExpectKnownValue(dataSourceAddr, tfjsonpath.New("building_blocks").AtSliceIndex(0).AtMapKey("metadata").AtMapKey("uuid"), xknownvalue.NotEmptyString(func(uuid string) error {
						if uuid != bbUuid {
							return fmt.Errorf("managed list returned block %q, expected the cross-workspace managed block %q", uuid, bbUuid)
						}
						return nil
					})),
				},
			},
			{
				// BUILDINGBLOCK_LIST alone → own/consumed scope; workspace A owns no blocks → empty.
				Config:          listConfig.String(),
				ConfigVariables: tfconfig.Variables{"apikey_client_id": &ownId, "apikey_client_secret": &ownSecret},
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(dataSourceAddr, tfjsonpath.New("building_blocks"), knownvalue.ListSizeExact(0)),
				},
			},
		}})
	})
}
