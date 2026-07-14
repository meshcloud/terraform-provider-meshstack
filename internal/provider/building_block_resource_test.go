package provider

import (
	"context"
	"fmt"
	"net/url"
	"os"
	"regexp"
	"testing"
	"time"

	tfconfig "github.com/hashicorp/terraform-plugin-testing/config"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/knownvalue"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
	"github.com/hashicorp/terraform-plugin-testing/statecheck"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	"github.com/hashicorp/terraform-plugin-testing/tfjsonpath"
	"github.com/stretchr/testify/require"

	"github.com/meshcloud/terraform-provider-meshstack/client"
	"github.com/meshcloud/terraform-provider-meshstack/internal/provider/acctest/testconfig"
	"github.com/meshcloud/terraform-provider-meshstack/internal/provider/acctest/xknownvalue"
)

// terraformTestdataRepoURL returns the clone URL of the committed bare git repo under
// testdata/tf-building-block (a single-commit no-op OpenTofu module) that the tf-block-runner clones
// to run terraform offline. In acceptance mode it serves the repo over git smart-HTTP (see
// git_http_server_test.go) and returns an http://127.0.0.1:<port>/... URL the runner can reach across
// containers -- a file:// URL cannot, since the runner has its own filesystem. In mock mode the value
// is never cloned, so a stable placeholder is returned without starting a server.
func terraformTestdataRepoURL(t *testing.T) string {
	t.Helper()
	if IsMockClientTest() {
		return "http://127.0.0.1:0/tf-building-block"
	}
	return gitHTTPRepoBaseURL(t) + "/tf-building-block"
}

// The subtests below are scenario flows rather than one-assertion-per-case tests: each walks a
// building block through a sequence of steps. Every flow runs in both modes; where an assertion
// only holds against the real backend the ConfigStateChecks are gated on IsMockClientTest(), and
// where a whole flow needs the real backend it lives in its own subtest that t.Skip()s in mock
// (see 07, 08's backend_rejections, 11). See the lock-step policy on IsMockClientTest.

// acceptanceClient builds a real meshStack API client from the test env vars.
// Only valid in acceptance mode (TF_ACC set); callers must guard with IsMockClientTest.
func acceptanceClient(t *testing.T) client.Client {
	t.Helper()
	rootUrl, err := url.Parse(os.Getenv(envKeyMeshstackEndpoint))
	require.NoError(t, err)
	auth := client.NewApiKeyAuthorization(os.Getenv(envKeyMeshstackApiKey), os.Getenv(envKeyMeshstackApiSecret))
	c, err := client.New(context.Background(), rootUrl, "acctest", auth)
	require.NoError(t, err)
	return c
}

// awaitBuildingBlockV1Succeeded polls the v1 building block until it reaches a final
// SUCCEEDED state. The meshstack_buildingblock (v1) resource has no wait_for_completion,
// so its run is still PENDING right after create; a subsequent move/replace that deletes
// it would otherwise hit the backend's non-final-status delete guard (409). Use as a
// step PreConfig before a move-from-v1 step.
func awaitBuildingBlockV1Succeeded(t *testing.T, uuid string) {
	t.Helper()
	ctx := context.Background()
	c := acceptanceClient(t)
	require.Eventuallyf(t, func() bool {
		bb, err := c.BuildingBlock.Read(ctx, uuid)
		return err == nil && bb != nil && bb.Status.Status == "SUCCEEDED"
	}, 120*time.Second, 3*time.Second, "v1 building block %s did not reach SUCCEEDED", uuid)
}

func TestAccBuildingBlock(t *testing.T) {
	t.Parallel()

	// 01_workspace_lifecycle: full workspace-BB life — create, import, in-place updates (display_name,
	// input value, content_hash), and a parent change that forces a replace. Step 8 (parent→Replace) is
	// mock-only: it asserts a provider-side plan decision (RequiresReplaceIf), and the real backend
	// rejects the synthetic parent-BB UUID before the plan can be observed.
	t.Run("01_workspace_lifecycle", func(t *testing.T) {
		config, buildingBlockAddr, buildingBlockDefinitionAddr, _ := testconfig.BBWorkspace(t)

		// Renaming display_name must be an in-place Update and must not change anything else.
		renamedConfig := config.WithFirstBlock(
			testconfig.Descend("spec", "display_name")(testconfig.SetRawExpr(`"my-workspace-building-block-renamed"`)),
		)
		updatedInputsConfig := renamedConfig.WithFirstBlock(
			testconfig.Descend("spec", "inputs", "environment", "value")(testconfig.SetRawExpr(`jsonencode("staging")`)),
		)
		// content_hash tracks the BBD version's content; setting it, then changing it, simulates the
		// BBD being updated and must trigger a rerun even though the version uuid is unchanged.
		contentHashV1Config := updatedInputsConfig.WithFirstBlock(
			testconfig.Descend("spec", "building_block_definition_version_ref")(testconfig.SetRawExpr(
				`merge(%s.version_latest, {content_hash = "v1"})`,
				buildingBlockDefinitionAddr,
			)),
		)
		contentHashV2Config := contentHashV1Config.WithFirstBlock(
			testconfig.Descend("spec", "building_block_definition_version_ref")(testconfig.SetRawExpr(
				`merge(%s.version_latest, {content_hash = "v2"})`,
				buildingBlockDefinitionAddr,
			)),
		)
		// Adding a parent without a version upgrade must force a replacement (RequiresReplaceIf →
		// DestroyBeforeCreate). Derived from the last config so only parent_building_blocks changes.
		withParentsConfig := contentHashV2Config.WithFirstBlock(
			testconfig.Descend("spec", "parent_building_blocks")(testconfig.SetRawExpr(
				`[{ buildingblock_uuid = "11111111-1111-1111-1111-111111111111", definition_uuid = "22222222-2222-2222-2222-222222222222" }]`,
			)),
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
					ConfigStateChecks: bbv3StateChecks(buildingBlockAddr, "my-workspace-building-block", bbv3SizeEnvInputChecks(buildingBlockAddr)...),
				},
				{
					// Import with verify. content_hash is json:"-" and never returned by the API;
					// wait_for_completion and purge_on_delete are config-only defaults — all excluded.
					ImportState:                          true,
					ImportStateVerify:                    true,
					ImportStateVerifyIdentifierAttribute: "metadata.uuid",
					ImportStateVerifyIgnore:              []string{"spec.building_block_definition_version_ref.content_hash", "wait_for_completion", "purge_on_delete", "timeouts.create", "timeouts.update", "timeouts.delete"},
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
					// The refreshed plan must be empty: unconfigured optional USER_INPUTs that the backend
					// echoes as null rows must not surface as drift. PlanOnly without ExpectNonEmptyPlan
					// asserts an empty plan. Runs in both modes — the mock materializes the same null rows,
					// so the check holds there too and we keep mock/acceptance behaviour in lock-step.
					Config:   config.String(),
					PlanOnly: true,
				},
				{
					// Rename only display_name → in-place Update, never Replace.
					Config: renamedConfig.String(),
					ConfigPlanChecks: resource.ConfigPlanChecks{
						PreApply: []plancheck.PlanCheck{
							plancheck.ExpectResourceAction(buildingBlockAddr.String(), plancheck.ResourceActionUpdate),
						},
					},
					ConfigStateChecks: []statecheck.StateCheck{
						statecheck.ExpectKnownValue(buildingBlockAddr.String(), tfjsonpath.New("spec").AtMapKey("display_name"), knownvalue.StringExact("my-workspace-building-block-renamed")),
					},
				},
				{
					// Change an input value → in-place Update.
					Config: updatedInputsConfig.String(),
					ConfigPlanChecks: resource.ConfigPlanChecks{
						PreApply: []plancheck.PlanCheck{
							plancheck.ExpectResourceAction(buildingBlockAddr.String(), plancheck.ResourceActionUpdate),
						},
					},
					ConfigStateChecks: []statecheck.StateCheck{
						statecheck.ExpectKnownValue(buildingBlockAddr.String(), tfjsonpath.New("spec").AtMapKey("inputs").AtMapKey("environment").AtMapKey("value"), knownvalue.StringExact(`"staging"`)),
					},
				},
				{
					// Set initial content_hash to track BBD version "v1".
					Config: contentHashV1Config.String(),
					ConfigPlanChecks: resource.ConfigPlanChecks{
						PreApply: []plancheck.PlanCheck{
							plancheck.ExpectResourceAction(buildingBlockAddr.String(), plancheck.ResourceActionUpdate),
						},
					},
					ConfigStateChecks: []statecheck.StateCheck{
						statecheck.ExpectKnownValue(buildingBlockAddr.String(), tfjsonpath.New("spec").AtMapKey("building_block_definition_version_ref").AtMapKey("content_hash"), knownvalue.StringExact("v1")),
					},
				},
				{
					// Bumping content_hash "v1"→"v2" simulates a BBD content update and triggers a rerun.
					Config: contentHashV2Config.String(),
					ConfigPlanChecks: resource.ConfigPlanChecks{
						PreApply: []plancheck.PlanCheck{
							plancheck.ExpectResourceAction(buildingBlockAddr.String(), plancheck.ResourceActionUpdate),
						},
					},
					ConfigStateChecks: []statecheck.StateCheck{
						statecheck.ExpectKnownValue(buildingBlockAddr.String(), tfjsonpath.New("spec").AtMapKey("building_block_definition_version_ref").AtMapKey("content_hash"), knownvalue.StringExact("v2")),
					},
				},
				{
					// Adding a parent while the version is unchanged must force a replacement. This is a
					// provider-side plan decision (RequiresReplaceIf → DestroyBeforeCreate); it runs
					// mock-only because the real backend rejects the synthetic parent-BB UUID before the
					// plan can be applied.
					SkipFunc: func() (bool, error) {
						return !IsMockClientTest(), nil
					},
					Config: withParentsConfig.String(),
					ConfigPlanChecks: resource.ConfigPlanChecks{
						PreApply: []plancheck.PlanCheck{
							plancheck.ExpectResourceAction(buildingBlockAddr.String(), plancheck.ResourceActionDestroyBeforeCreate),
						},
					},
				},
			},
		})
	})

	// 02_tenant covers the tenant-targeted create + import-with-verify path (the tenant analogue of
	// the 01 create/import steps). Tenant target_ref uses uuid (not name). The BBD uses the terraform
	// implementation (the real tf-block-runner clones the local bare repo and runs OpenTofu in
	// acceptance) and declares a STRING-typed sensitive api_key USER_INPUT. The sensitive-hash check
	// holds in both modes — the mock hashes any sensitive plaintext just like the backend — so no
	// mock/acceptance branch is needed here.
	t.Run("02_tenant", func(t *testing.T) {
		config, buildingBlockAddr, _ := testconfig.BBTenant(t, terraformTestdataRepoURL(t))

		// api_key (sensitive STRING USER_INPUT) surfaces as a non-empty hash in all_inputs in both modes.
		sensitiveInputChecks := append(bbv3SizeEnvInputChecks(buildingBlockAddr),
			statecheck.ExpectKnownValue(buildingBlockAddr.String(),
				tfjsonpath.New("all_inputs").AtMapKey("api_key").AtMapKey("sensitive").AtMapKey("secret_hash"),
				xknownvalue.NotEmptyString()),
		)
		if !IsMockClientTest() {
			// Acceptance-only proof of end-to-end decryption: the real tf-block-runner decrypts the
			// sensitive api_key and the module echoes the plaintext back as the non-sensitive
			// `api_key_echo` output (declared in the BBD), surfaced here on status.outputs. The value is
			// JSON-encoded, so a STRING output is quoted. The mock neither runs OpenTofu nor produces
			// outputs, so this check is gated.
			sensitiveInputChecks = append(sensitiveInputChecks,
				statecheck.ExpectKnownValue(buildingBlockAddr.String(),
					tfjsonpath.New("status").AtMapKey("outputs").AtMapKey("api_key_echo").AtMapKey("value"),
					knownvalue.StringExact(`"super-secret-api-key"`)),
			)
		}

		ApplyAndTest(t, resource.TestCase{
			Steps: []resource.TestStep{
				{
					Config: config.String(),
					ConfigPlanChecks: resource.ConfigPlanChecks{
						PreApply: []plancheck.PlanCheck{
							plancheck.ExpectResourceAction(buildingBlockAddr.String(), plancheck.ResourceActionCreate),
						},
					},
					ConfigStateChecks: bbv3StateChecks(buildingBlockAddr, "my-tenant-building-block", sensitiveInputChecks...),
				},
				{
					// Import with verify; ignore config-only fields. secret_version is excluded because the
					// backend returns the sensitive input as a hash on read, so the supplied version is not
					// recoverable on import (same reason content_hash is excluded).
					ImportState:                          true,
					ImportStateVerify:                    true,
					ImportStateVerifyIdentifierAttribute: "metadata.uuid",
					ImportStateVerifyIgnore: []string{
						"spec.building_block_definition_version_ref.content_hash",
						"spec.inputs.api_key.sensitive.secret_version",
						"wait_for_completion",
						"purge_on_delete",
						"timeouts.create",
						"timeouts.update",
						"timeouts.delete",
					},
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

	// 03_workspace_moved_from_v2 guards the v2→v3 migration: a `moved` block from
	// meshstack_building_block_v2 to v3 must plan as an in-place Update, never a destroy+recreate.
	// moveFromV2 leaves target_ref/version_ref to be filled by the post-move refresh-Read, so the
	// RequiresReplace modifiers see equal values and do not fire.
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
		v3Config := testconfig.Resource{Name: "building_block", Suffix: "_01_workspace"}.Config(t).WithFirstBlock(
			testconfig.ExtractAddress(&v3Addr),
			testconfig.Descend("spec", "building_block_definition_version_ref")(testconfig.SetAddr(buildingBlockDefinitionAddr, "version_latest")),
			testconfig.Descend("spec", "target_ref")(testconfig.SetAddr(workspaceAddr, "ref")),
		).Join(workspaceConfig, buildingBlockDefinitionConfig)

		// The moved-block source/target addresses are fixed (resource labels are not randomized), so the
		// test-support file hard-codes them directly instead of substituting via SetAddr.
		movedConfig := testconfig.Resource{Name: "building_block"}.TestSupportConfig(t, "_moved_from_v2")

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
					// The move must plan as an in-place Update; a regression to Replace fails here.
					Config: v3Config.Join(movedConfig).String(),
					ConfigPlanChecks: resource.ConfigPlanChecks{
						PreApply: []plancheck.PlanCheck{
							plancheck.ExpectResourceAction(v3Addr.String(), plancheck.ResourceActionUpdate),
						},
					},
					ConfigStateChecks: bbv3StateChecks(v3Addr, "my-workspace-building-block", bbv3SizeEnvInputChecks(v3Addr)...),
				},
			},
		})
	})

	// 04_tenant_moved_from_v1 guards the v1→v3 migration: a `moved` block from the legacy
	// meshstack_buildingblock (v1) to v3 must plan as an in-place Update, never a destroy+recreate
	// (recreating a live tenant BB is destructive). moveFromV1 leaves target_ref/version_ref unset so
	// the post-move refresh-Read fills them from the live DTO before the RequiresReplace modifiers
	// evaluate. The move step's PreConfig awaits the v1 run's SUCCEEDED first (see
	// awaitBuildingBlockV1Succeeded).
	t.Run("04_tenant_moved_from_v1", func(t *testing.T) {

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

		// Dedicated migration fixtures (manual impl, no sensitive inputs): the v1 legacy resource
		// cannot carry sensitive inputs, so this test stays decoupled from the terraform + sensitive
		// _02_tenant showcase, which both the v1 and v3 sides would otherwise have to satisfy.
		var buildingBlockDefinitionAddr testconfig.Traversal
		buildingBlockDefinitionConfig := testconfig.Resource{Name: "building_block", Suffix: "_tenant_migration"}.TestSupportConfig(t, "_bbd").WithFirstBlock(
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
		v3Config := testconfig.Resource{Name: "building_block", Suffix: "_tenant_migration"}.TestSupportConfig(t, "_bb").WithFirstBlock(
			testconfig.ExtractAddress(&v3Addr),
			testconfig.Descend("spec", "building_block_definition_version_ref")(testconfig.SetAddr(buildingBlockDefinitionAddr, "version_latest")),
			testconfig.Descend("spec", "target_ref")(testconfig.SetAddr(tenantAddr, "ref")),
		).Join(workspaceConfig, projectConfig, platformConfig, landingZoneConfig, tenantConfig, buildingBlockDefinitionConfig)

		// The moved-block source/target addresses are fixed (resource labels are not randomized), so the
		// test-support file hard-codes them directly instead of substituting via SetAddr.
		movedConfig := testconfig.Resource{Name: "building_block"}.TestSupportConfig(t, "_moved_from_v1")

		var v1Uuid string
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
						statecheck.ExpectKnownValue(v1Addr.String(), tfjsonpath.New("metadata").AtMapKey("uuid"), xknownvalue.NotEmptyString(func(uuid string) error {
							v1Uuid = uuid
							return nil
						})),
					},
				},
				{
					// Await the v1 run's SUCCEEDED state before the move (acceptance-only); otherwise
					// the move-Update would hit the backend's completed-state guard (409).
					PreConfig: func() {
						if !IsMockClientTest() {
							awaitBuildingBlockV1Succeeded(t, v1Uuid)
						}
					},
					Config: v3Config.Join(movedConfig).String(),
					// Verified against a live backend (Plan: 0 add, 1 change, 0 destroy; metadata.uuid
					// preserved across the move). A regression to Replace fails here.
					ConfigPlanChecks: resource.ConfigPlanChecks{
						PreApply: []plancheck.PlanCheck{
							plancheck.ExpectResourceAction(v3Addr.String(), plancheck.ResourceActionUpdate),
						},
					},
					ConfigStateChecks: bbv3StateChecks(v3Addr, "my-tenant-building-block"),
				},
			},
		})
	})

	// 05_operator_inputs walks a platform-operator input (`size`, PLATFORM_OPERATOR_MANUAL_INPUT) through
	// its lifecycle on a cross-workspace BB: create without it (parks WAITING_FOR_OPERATOR_INPUT), resume
	// via a PUT that supplies it (provider must poll THROUGH the transient WAITING — see
	// TestAwaitRunPollsThroughWaiting), update a consumer input, then upgrade to a v2 BBD that adds a
	// defaulted operator input. All steps run in both modes; only two assertions are backend-gated (the
	// mock has no WAITING state, and does not materialize the v2 default `tier`). A
	// MANAGED_BUILDINGBLOCK_SAVE key is minted to prove the provider accepts that authority; its
	// cross-workspace authorization is covered on the backend by MeshBuildingBlockManagedSaveScenarios.
	t.Run("05_operator_inputs", func(t *testing.T) {
		// Workspace A owns the BBD (which declares `size` as PLATFORM_OPERATOR_MANUAL_INPUT) and
		// the API key used to set the operator input.
		workspaceConfig, workspaceAddr := testconfig.Workspace(t)
		var buildingBlockDefinitionAddr testconfig.Traversal
		buildingBlockDefinitionConfig := testconfig.Resource{Name: "building_block", Suffix: "_03_operator_inputs"}.TestSupportConfig(t, "").WithFirstBlock(
			testconfig.ExtractAddress(&buildingBlockDefinitionAddr),
			testconfig.OwnedByWorkspace(workspaceAddr),
		)

		// Workspace B is the consumer: the building block lives here, across the workspace boundary
		// from the definition owner.
		var otherWorkspaceAddr testconfig.Traversal
		otherWorkspaceConfig, _ := testconfig.Workspace(t)
		otherWorkspaceConfig = otherWorkspaceConfig.WithFirstBlock(
			testconfig.RenameKey("other"),
			testconfig.ExtractAddress(&otherWorkspaceAddr),
		)

		// Key owned by workspace A (the definition owner). MANAGED_BUILDINGBLOCK_SAVE is the capability
		// under test: it lets the key create and update building blocks consuming its definition in
		// other workspaces (the key deliberately has no ADM_BUILDINGBLOCK_SAVE, so every create/update
		// below is authorized purely by the MANAGED scope). ADM_BUILDINGBLOCK_DELETE is present only so
		// the meshstack-other provider can tear the cross-workspace block down at end of test: there is
		// no MANAGED_BUILDINGBLOCK_DELETE authority, so a platform operator can save but not delete a
		// block it doesn't own — delete is not the capability being exercised here.
		apiKeyConfig, apiKeyAddr := testconfig.ApiKey(t, workspaceAddr)
		apiKeyConfig = apiKeyConfig.WithFirstBlock(
			testconfig.Descend("spec", "permissions")(testconfig.SetRawExpr(`["MANAGED_BUILDINGBLOCK_SAVE", "MANAGED_BUILDINGBLOCK_LIST", "ADM_BUILDINGBLOCK_DELETE"]`)),
		)

		// Step 1 sets up the infrastructure and mints the key (default/admin provider).
		step1Config := workspaceConfig.Join(buildingBlockDefinitionConfig, otherWorkspaceConfig, apiKeyConfig)

		// Steps 2+ keep the meshstack-other provider alias configured (the MANAGED key minted above); the
		// block itself is admin-managed, so every step passes the key credentials as config variables.
		otherProviderConfig := testconfig.Resource{Name: "building_block"}.TestSupportConfig(t, "_other_provider")

		// v2 of the definition adds a defaulted platform-operator input `tier`. Re-draft (new v2 draft;
		// version_latest_release still v1) then re-release (v2 released) mirror the version dance in 06, so
		// the upgrade steps below prove the backend applies an operator-input default on upgrade.
		withTier := func(c testconfig.Config) testconfig.Config {
			return c.WithFirstBlock(testconfig.Descend("version_spec", "inputs", "tier")(testconfig.SetRawExpr(`{
  display_name           = "Tier"
  description            = "A defaulted platform operator input"
  type                   = "STRING"
  assignment_type        = "PLATFORM_OPERATOR_MANUAL_INPUT"
  default_value          = jsonencode("bronze")
  updateable_by_consumer = true
}`)))
		}
		bbdV2Draft := withTier(buildingBlockDefinitionConfig).WithFirstBlock(testconfig.Descend("version_spec", "draft")(testconfig.SetRawExpr("true")))
		bbdV2Released := withTier(buildingBlockDefinitionConfig).WithFirstBlock(testconfig.Descend("version_spec", "draft")(testconfig.SetRawExpr("false")))

		// Reuses the workspace example (resource_01_workspace.tf); the operator BBD above reinterprets its
		// `size` input as a platform-operator input. Pinning version_latest_release lets the v2 release drive
		// an in-place upgrade. buildBB rebuilds the block + its dependencies for a given definition version and
		// input set, so each step can vary the inputs (operator-input WAITING/resume) and the version (upgrade).
		var buildingBlockAddr testconfig.Traversal
		buildBB := func(bbdConfig testconfig.Config, inputsExpr string) testconfig.Config {
			return testconfig.Resource{Name: "building_block", Suffix: "_01_workspace"}.Config(t).WithFirstBlock(
				testconfig.ExtractAddress(&buildingBlockAddr),
				testconfig.Descend("spec", "building_block_definition_version_ref")(testconfig.SetRawExpr(`{ uuid = %s }`, buildingBlockDefinitionAddr.Join("version_latest_release", "uuid"))),
				testconfig.Descend("spec", "target_ref")(testconfig.SetAddr(otherWorkspaceAddr, "ref")),
				testconfig.Descend("spec", "inputs")(testconfig.SetRawExpr("%s", inputsExpr)),
				// Admin-managed (default provider): creating cross-workspace and setting an operator input
				// require ADM_BUILDINGBLOCK_SAVE. depends_on keeps the minted key alive for teardown.
				testconfig.Descend("depends_on")(testconfig.SetRawExpr("[%s]", apiKeyAddr)),
			).Join(workspaceConfig, otherWorkspaceConfig, apiKeyConfig, bbdConfig, otherProviderConfig)
		}

		const inputsNoSize = `{
  name        = { value = jsonencode("my-name") }
  environment = { value = jsonencode("dev") }
}`
		const inputsWithSize = `{
  name        = { value = jsonencode("my-name") }
  size        = { value = jsonencode(16) }
  environment = { value = jsonencode("dev") }
}`
		const inputsRenamed = `{
  name        = { value = jsonencode("updated-name") }
  size        = { value = jsonencode(16) }
  environment = { value = jsonencode("dev") }
}`

		// Pre-build the step configs so buildingBlockAddr is populated before the state checks reference it.
		createConfig := buildBB(buildingBlockDefinitionConfig, inputsNoSize)
		suppliedConfig := buildBB(buildingBlockDefinitionConfig, inputsWithSize)
		renamedConfig := buildBB(buildingBlockDefinitionConfig, inputsRenamed)
		redraftConfig := buildBB(bbdV2Draft, inputsRenamed)
		upgradedConfig := buildBB(bbdV2Released, inputsRenamed)

		// On a real backend the operator-input-less create parks WAITING_FOR_OPERATOR_INPUT; the mock has
		// no such state and short-circuits the create to a terminal status.
		createStatusCheck := statecheck.ExpectKnownValue(buildingBlockAddr.String(), tfjsonpath.New("status").AtMapKey("status"), knownvalue.StringExact("WAITING_FOR_OPERATOR_INPUT"))
		if IsMockClientTest() {
			createStatusCheck = statecheck.ExpectKnownValue(buildingBlockAddr.String(), tfjsonpath.New("status").AtMapKey("status"), xknownvalue.NotEmptyString())
		}

		// Post-upgrade checks: the block reaches SUCCEEDED in both modes. The `tier` assertion is
		// acceptance-only — applying a defaulted operator input on upgrade is backend behaviour the
		// mock does not reproduce (it neither runs the block nor materializes the added default).
		upgradedChecks := []statecheck.StateCheck{
			statecheck.ExpectKnownValue(buildingBlockAddr.String(), tfjsonpath.New("status").AtMapKey("status"), knownvalue.StringExact("SUCCEEDED")),
		}
		if !IsMockClientTest() {
			upgradedChecks = append(upgradedChecks,
				statecheck.ExpectKnownValue(buildingBlockAddr.String(), tfjsonpath.New("all_inputs").AtMapKey("tier").AtMapKey("value"), knownvalue.StringExact(`"bronze"`)),
			)
		}

		var apiKeyClientId, apiKeyClientSecret lazyVariable
		creds := tfconfig.Variables{
			"apikey_client_id":     &apiKeyClientId,
			"apikey_client_secret": &apiKeyClientSecret,
		}
		ApplyAndTest(t, resource.TestCase{
			Steps: []resource.TestStep{
				{
					// Mint the MANAGED key (default/admin provider) and capture its credentials.
					Config: step1Config.String(),
					ConfigStateChecks: []statecheck.StateCheck{
						statecheck.ExpectKnownValue(apiKeyAddr.String(), tfjsonpath.New("status").AtMapKey("client_id"), xknownvalue.NotEmptyString(func(clientId string) error {
							apiKeyClientId = lazyVariable(clientId)
							return nil
						})),
						statecheck.ExpectKnownValue(apiKeyAddr.String(), tfjsonpath.New("status").AtMapKey("client_secret"), xknownvalue.NotEmptyString(func(clientSecret string) error {
							apiKeyClientSecret = lazyVariable(clientSecret)
							return nil
						})),
					},
				},
				{
					// Create without the operator input → on a real backend the block parks
					// WAITING_FOR_OPERATOR_INPUT (the provider surfaces a warning, not an error).
					Config:          createConfig.String(),
					ConfigVariables: creds,
					ConfigPlanChecks: resource.ConfigPlanChecks{
						PreApply: []plancheck.PlanCheck{
							plancheck.ExpectResourceAction(buildingBlockAddr.String(), plancheck.ResourceActionCreate),
						},
					},
					ConfigStateChecks: []statecheck.StateCheck{
						statecheck.ExpectKnownValue(buildingBlockAddr.String(), tfjsonpath.New("spec").AtMapKey("inputs").AtMapKey("name").AtMapKey("value"), knownvalue.StringExact(`"my-name"`)),
						createStatusCheck,
					},
				},
				{
					// Supplying `size` via PUT resumes provisioning; the provider must poll THROUGH the
					// transient WAITING to the resulting SUCCEEDED run instead of returning on the stale WAITING.
					Config:          suppliedConfig.String(),
					ConfigVariables: creds,
					ConfigPlanChecks: resource.ConfigPlanChecks{
						PreApply: []plancheck.PlanCheck{
							plancheck.ExpectResourceAction(buildingBlockAddr.String(), plancheck.ResourceActionUpdate),
						},
					},
					ConfigStateChecks: []statecheck.StateCheck{
						statecheck.ExpectKnownValue(buildingBlockAddr.String(), tfjsonpath.New("all_inputs").AtMapKey("size").AtMapKey("value"), knownvalue.StringExact("16")),
						statecheck.ExpectKnownValue(buildingBlockAddr.String(), tfjsonpath.New("status").AtMapKey("status"), knownvalue.StringExact("SUCCEEDED")),
						statecheck.ExpectKnownValue(buildingBlockAddr.String(), tfjsonpath.New("status").AtMapKey("latest_run_uuid"), xknownvalue.NotEmptyString()),
					},
				},
				{
					// Changing a consumer input is an in-place Update; the operator input stays put.
					Config:          renamedConfig.String(),
					ConfigVariables: creds,
					ConfigPlanChecks: resource.ConfigPlanChecks{
						PreApply: []plancheck.PlanCheck{
							plancheck.ExpectResourceAction(buildingBlockAddr.String(), plancheck.ResourceActionUpdate),
						},
					},
					ConfigStateChecks: []statecheck.StateCheck{
						statecheck.ExpectKnownValue(buildingBlockAddr.String(), tfjsonpath.New("spec").AtMapKey("inputs").AtMapKey("name").AtMapKey("value"), knownvalue.StringExact(`"updated-name"`)),
						statecheck.ExpectKnownValue(buildingBlockAddr.String(), tfjsonpath.New("all_inputs").AtMapKey("size").AtMapKey("value"), knownvalue.StringExact("16")),
					},
				},
				{
					// Re-draft the BBD → v2 draft (adds the defaulted operator input);
					// version_latest_release still resolves to v1, so the block is a no-op.
					Config:          redraftConfig.String(),
					ConfigVariables: creds,
					ConfigPlanChecks: resource.ConfigPlanChecks{
						PreApply: []plancheck.PlanCheck{
							plancheck.ExpectResourceAction(buildingBlockDefinitionAddr.String(), plancheck.ResourceActionUpdate),
							plancheck.ExpectResourceAction(buildingBlockAddr.String(), plancheck.ResourceActionNoop),
						},
					},
				},
				{
					// Release v2 and upgrade the cross-workspace block to it. v2 adds a
					// defaulted operator input the config does not supply; the backend applies the default on
					// upgrade, so the block reaches SUCCEEDED (not WAITING) and surfaces it in all_inputs.
					Config:          upgradedConfig.String(),
					ConfigVariables: creds,
					ConfigPlanChecks: resource.ConfigPlanChecks{
						PreApply: []plancheck.PlanCheck{
							plancheck.ExpectResourceAction(buildingBlockDefinitionAddr.String(), plancheck.ResourceActionUpdate),
							plancheck.ExpectResourceAction(buildingBlockAddr.String(), plancheck.ResourceActionUpdate),
						},
					},
					ConfigStateChecks: upgradedChecks,
				},
			},
		})
	})

	// 06_sensitive_inputs_and_upgrade: one BBD with STRING (api_key), CODE (script) and STATIC
	// (static_secret) sensitive inputs; the BB is created, upgraded across a BBD re-draft/re-release,
	// then rotated. All steps run in both modes (the mock supports the version dance and hashes any
	// sensitive plaintext); only the STATIC hash and the SUCCEEDED create status are backend-gated. The
	// version ref pins version_latest_release.uuid so the BB stays on the released version while a draft
	// exists; on the upgrade PUT the sensitive hash sentinel must preserve the secret, not corrupt it.
	t.Run("06_sensitive_inputs_and_upgrade", func(t *testing.T) {
		workspaceConfig, workspaceAddr := testconfig.Workspace(t)
		exampleResource := testconfig.Resource{Name: "building_block", Suffix: "_04_sensitive_user_input"}

		var bbdAddr testconfig.Traversal
		bbdV1Released := exampleResource.TestSupportConfig(t, "_bbd").WithFirstBlock(
			testconfig.ExtractAddress(&bbdAddr),
			testconfig.OwnedByWorkspace(workspaceAddr),
			// Point the terraform implementation at the committed bare repo served over loopback git
			// smart-HTTP so the real tf-block-runner clones and runs OpenTofu offline. In mock mode this
			// value is unused. The static example URL in the .tf is only a docs placeholder.
			testconfig.Descend("version_spec", "implementation", "terraform", "repository_url")(
				testconfig.SetRawExpr("%q", terraformTestdataRepoURL(t)),
			),
		)
		// Re-draft (creates a v2 draft; version_latest_release still points to v1) and re-release
		// (releases v2). The mock supports the version dance, so the upgrade steps run in both modes.
		bbdV2Draft := bbdV1Released.WithFirstBlock(
			testconfig.Descend("version_spec", "draft")(testconfig.SetRawExpr("true")),
		)
		bbdV2Released := bbdV1Released.WithFirstBlock(
			testconfig.Descend("version_spec", "draft")(testconfig.SetRawExpr("false")),
		)

		var bbAddr testconfig.Traversal
		// Build a BB pinned to the BBD's latest released version, supplying api_key + script secrets
		// and pinning api_key.secret_version so the rotation step can bump it.
		buildBBConfig := func(bbdConfig testconfig.Config) testconfig.Config {
			return exampleResource.TestSupportConfig(t, "").WithFirstBlock(
				testconfig.ExtractAddress(&bbAddr),
				testconfig.Descend("spec", "building_block_definition_version_ref")(
					testconfig.SetRawExpr(`{ uuid = %s }`, bbdAddr.Join("version_latest_release", "uuid")),
				),
				testconfig.Descend("spec", "target_ref")(testconfig.SetAddr(workspaceAddr, "ref")),
				testconfig.Descend("spec", "inputs", "api_key", "sensitive", "secret_version")(testconfig.SetRawExpr(`"1"`)),
			).Join(workspaceConfig, bbdConfig)
		}

		configV1 := buildBBConfig(bbdV1Released)         // BBD v1 released + BB on v1
		configRedraft := buildBBConfig(bbdV2Draft)       // BBD v2 draft + BB still on v1
		configV2Released := buildBBConfig(bbdV2Released) // BBD v2 released + BB upgrading to v2
		// Rotate api_key (new secret_value + bumped secret_version) on the post-upgrade config (the BB
		// is on v2 in both modes by this point) — still a rotation.
		rotatedConfig := configV2Released.WithFirstBlock(
			testconfig.Descend("spec", "inputs", "api_key", "sensitive", "secret_value")(testconfig.SetRawExpr(`"rotated-api-key"`)),
			testconfig.Descend("spec", "inputs", "api_key", "sensitive", "secret_version")(testconfig.SetRawExpr(`"2"`)),
		)

		var lastRunUuid string
		captureRunUuid := xknownvalue.NotEmptyString(func(v string) error {
			lastRunUuid = v
			return nil
		})

		// Create-step checks that hold in both modes: api_key (STRING) and script (CODE) sensitive
		// hashes plus the run uuid. The mock hashes any sensitive plaintext just like the backend, so
		// both USER_INPUT hashes surface in mock too — only the STATIC hash and the SUCCEEDED status
		// (below) genuinely need the real backend.
		createChecks := []statecheck.StateCheck{
			statecheck.ExpectKnownValue(bbAddr.String(), tfjsonpath.New("metadata").AtMapKey("uuid"), xknownvalue.NotEmptyString()),
			statecheck.ExpectKnownValue(bbAddr.String(),
				tfjsonpath.New("all_inputs").AtMapKey("api_key").AtMapKey("sensitive").AtMapKey("secret_hash"),
				xknownvalue.NotEmptyString()),
			statecheck.ExpectKnownValue(bbAddr.String(),
				tfjsonpath.New("all_inputs").AtMapKey("script").AtMapKey("sensitive").AtMapKey("secret_hash"),
				xknownvalue.NotEmptyString()),
			statecheck.ExpectKnownValue(bbAddr.String(), tfjsonpath.New("status").AtMapKey("latest_run_uuid"), captureRunUuid),
		}
		if !IsMockClientTest() {
			createChecks = append(createChecks,
				// STATIC sensitive input: the mock does not resolve STATIC inputs from the BBD.
				statecheck.ExpectKnownValue(bbAddr.String(),
					tfjsonpath.New("all_inputs").AtMapKey("static_secret").AtMapKey("sensitive").AtMapKey("secret_hash"),
					xknownvalue.NotEmptyString()),
				// Acceptance-only: with wait_for_completion (default true) the create apply blocks until
				// the run is terminal. Against the real tf-block-runner (cloning the local bare repo and
				// running OpenTofu) this proves the run actually reached SUCCEEDED — the no-op manual
				// runner used to complete it trivially without running terraform.
				statecheck.ExpectKnownValue(bbAddr.String(),
					tfjsonpath.New("status").AtMapKey("status"),
					knownvalue.StringExact("SUCCEEDED")),
			)
		}

		ApplyAndTest(t, resource.TestCase{
			Steps: []resource.TestStep{
				{
					// Step 1: create BBD v1 + BB on v1. Sensitive inputs surface as hashes in all_inputs.
					Config: configV1.String(),
					ConfigPlanChecks: resource.ConfigPlanChecks{
						PreApply: []plancheck.PlanCheck{
							plancheck.ExpectResourceAction(bbAddr.String(), plancheck.ResourceActionCreate),
						},
					},
					ConfigStateChecks: createChecks,
				},
				{
					// Step 2: import with verify. Besides content_hash and the
					// config-only flags, secret_version is excluded: the backend returns it as a hash
					// of the secret on read, so the user-supplied version number is not recoverable on
					// import (the same reason content_hash is excluded).
					ImportState:                          true,
					ImportStateVerify:                    true,
					ImportStateVerifyIdentifierAttribute: "metadata.uuid",
					ImportStateVerifyIgnore: []string{
						"spec.building_block_definition_version_ref.content_hash",
						"spec.inputs.api_key.sensitive.secret_version",
						"wait_for_completion",
						"purge_on_delete",
						"timeouts.create",
						"timeouts.update",
						"timeouts.delete",
					},
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
					// Step 3: re-draft the BBD → v2 draft. version_latest_release
					// still resolves to v1, so the BB plan is a no-op.
					Config: configRedraft.String(),
					ConfigPlanChecks: resource.ConfigPlanChecks{
						PreApply: []plancheck.PlanCheck{
							plancheck.ExpectResourceAction(bbdAddr.String(), plancheck.ResourceActionUpdate),
							plancheck.ExpectResourceAction(bbAddr.String(), plancheck.ResourceActionNoop),
						},
					},
				},
				{
					// Step 4: release BBD v2 + upgrade the BB to v2 in one apply. The
					// sensitive api_key is echoed as its hash sentinel and the secret must survive.
					Config: configV2Released.String(),
					ConfigPlanChecks: resource.ConfigPlanChecks{
						PreApply: []plancheck.PlanCheck{
							plancheck.ExpectResourceAction(bbdAddr.String(), plancheck.ResourceActionUpdate),
							plancheck.ExpectResourceAction(bbAddr.String(), plancheck.ResourceActionUpdate),
						},
					},
					ConfigStateChecks: []statecheck.StateCheck{
						statecheck.ExpectKnownValue(bbAddr.String(), tfjsonpath.New("metadata").AtMapKey("uuid"), xknownvalue.NotEmptyString()),
						// Sensitive hash still present after upgrade (secret not corrupted).
						statecheck.ExpectKnownValue(bbAddr.String(),
							tfjsonpath.New("all_inputs").AtMapKey("api_key").AtMapKey("sensitive").AtMapKey("secret_hash"),
							xknownvalue.NotEmptyString()),
						// Re-capture the run uuid so the rotation step below compares against the upgrade run.
						statecheck.ExpectKnownValue(bbAddr.String(), tfjsonpath.New("status").AtMapKey("latest_run_uuid"), captureRunUuid),
					},
				},
				{
					// Step 5: post-upgrade plan must be empty — no spurious rerun and
					// no phantom-input drift.
					Config:   configV2Released.String(),
					PlanOnly: true,
				},
				{
					// Step 6: rotate api_key (bumped secret_version). Rotation is invisible to the rerun
					// predicate and must be detected via the changed secret_version, so this is an
					// in-place Update and latest_run_uuid must change vs. the previous run.
					Config: rotatedConfig.String(),
					ConfigPlanChecks: resource.ConfigPlanChecks{
						PreApply: []plancheck.PlanCheck{
							plancheck.ExpectResourceAction(bbAddr.String(), plancheck.ResourceActionUpdate),
						},
					},
					ConfigStateChecks: []statecheck.StateCheck{
						statecheck.ExpectKnownValue(bbAddr.String(),
							tfjsonpath.New("status").AtMapKey("latest_run_uuid"),
							xknownvalue.NotEmptyString(func(v string) error {
								if v == lastRunUuid {
									return fmt.Errorf("expected a rerun: latest_run_uuid should change after secret rotation, but stayed %q", v)
								}
								return nil
							})),
					},
				},
			},
		})
	})

	// 07_non_updateable_rejected is a cross-workspace permission test: a consumer in another workspace
	// (using a scoped API key) must be rejected when it tries to change an input the BBD marks
	// non-updateable-by-consumer. Acceptance-only — it needs real permission boundaries and a second
	// provider alias, which the mock cannot model.
	t.Run("07_non_updateable_rejected", func(t *testing.T) {
		if IsMockClientTest() {
			t.Skip("cross-workspace test requires real permission boundaries")
		}

		// Step 1 config: admin creates workspace + BBD + other workspace + API key.
		workspaceConfig, workspaceAddr := testconfig.Workspace(t)
		var buildingBlockDefinitionAddr testconfig.Traversal
		buildingBlockDefinitionConfig := testconfig.Resource{Name: "building_block", Suffix: "_07_non_updateable"}.TestSupportConfig(t, "").WithFirstBlock(
			testconfig.ExtractAddress(&buildingBlockDefinitionAddr),
			testconfig.OwnedByWorkspace(workspaceAddr),
		)

		var otherWorkspaceAddr testconfig.Traversal
		otherWorkspaceConfig, _ := testconfig.Workspace(t)
		otherWorkspaceConfig = otherWorkspaceConfig.WithFirstBlock(
			testconfig.RenameKey("other"),
			testconfig.ExtractAddress(&otherWorkspaceAddr),
		)
		apiKeyConfig, apiKeyAddr := testconfig.ApiKey(t, otherWorkspaceAddr)
		apiKeyConfig = apiKeyConfig.WithFirstBlock(
			testconfig.Descend("spec", "permissions")(testconfig.SetRawExpr(`["BUILDINGBLOCK_SAVE", "BUILDINGBLOCK_LIST", "BUILDINGBLOCK_DELETE"]`)),
		)

		step1Config := workspaceConfig.Join(buildingBlockDefinitionConfig, otherWorkspaceConfig, apiKeyConfig)

		// Step 2 config: the "other" provider creates a BB with consumer-only inputs.
		otherProviderConfig := testconfig.Resource{Name: "building_block"}.TestSupportConfig(t, "_other_provider")

		// Reuses the workspace example (resource_01_workspace.tf) 1:1; the BBD above marks its
		// `environment` input non-updateable-by-consumer.
		var buildingBlockAddr testconfig.Traversal
		bbConfig := testconfig.Resource{Name: "building_block", Suffix: "_01_workspace"}.Config(t).WithFirstBlock(
			testconfig.ExtractAddress(&buildingBlockAddr),
			testconfig.Descend("spec", "building_block_definition_version_ref")(testconfig.SetAddr(buildingBlockDefinitionAddr, "version_latest")),
			testconfig.Descend("spec", "target_ref")(testconfig.SetAddr(otherWorkspaceAddr, "ref")),
			testconfig.Descend("provider")(testconfig.SetRawExpr("meshstack-other")),
			// depends_on ensures the BB is destroyed before the API key the other provider needs.
			testconfig.Descend("depends_on")(testconfig.SetRawExpr("[%s]", apiKeyAddr)),
		)

		step2Config := bbConfig.Join(step1Config, otherProviderConfig)

		// Step 3 config: try to change the non-updateable input (must fail).
		step3Config := step2Config.WithFirstBlock(
			testconfig.Descend("spec", "inputs", "environment")(testconfig.SetRawExpr(`{ value = jsonencode("staging") }`)),
		)

		var apiKeyClientId, apiKeyClientSecret lazyVariable
		ApplyAndTest(t, resource.TestCase{
			Steps: []resource.TestStep{
				{
					Config: step1Config.String(),
					ConfigStateChecks: []statecheck.StateCheck{
						statecheck.ExpectKnownValue(apiKeyAddr.String(), tfjsonpath.New("status").AtMapKey("client_id"), xknownvalue.NotEmptyString(func(clientId string) error {
							apiKeyClientId = lazyVariable(clientId)
							return nil
						})),
						statecheck.ExpectKnownValue(apiKeyAddr.String(), tfjsonpath.New("status").AtMapKey("client_secret"), xknownvalue.NotEmptyString(func(clientSecret string) error {
							apiKeyClientSecret = lazyVariable(clientSecret)
							return nil
						})),
					},
				},
				{
					Config: step2Config.String(),
					ConfigVariables: tfconfig.Variables{
						"apikey_client_id":     &apiKeyClientId,
						"apikey_client_secret": &apiKeyClientSecret,
					},
					ConfigPlanChecks: resource.ConfigPlanChecks{
						PreApply: []plancheck.PlanCheck{
							plancheck.ExpectResourceAction(buildingBlockAddr.String(), plancheck.ResourceActionCreate),
						},
					},
					ConfigStateChecks: []statecheck.StateCheck{
						statecheck.ExpectKnownValue(buildingBlockAddr.String(), tfjsonpath.New("spec").AtMapKey("inputs").AtMapKey("environment").AtMapKey("value"), knownvalue.StringExact(`"dev"`)),
					},
				},
				{
					Config: step3Config.String(),
					ConfigVariables: tfconfig.Variables{
						"apikey_client_id":     &apiKeyClientId,
						"apikey_client_secret": &apiKeyClientSecret,
					},
					ExpectError: regexp.MustCompile("you don't have sufficient permissions"),
				},
			},
		})
	})

	// 08_validation_rejects collects the plan-time and create-time rejection checks over the workspace
	// BBWorkspace config: a target_ref whose kind and identifier disagree, and a STATIC BBD input
	// wrongly assigned as a customer input. It splits along the mock/acceptance line into two subtests
	// (each a single ApplyAndTest), so no per-step mode gate is needed:
	//   - provider_side_validators runs in BOTH modes: the target_ref kind/identifier mismatch is
	//     rejected by a provider-side validator before any backend call, so the mock exercises it too.
	//   - backend_rejections runs in ACCEPTANCE only (it skips itself in mock): the STATIC-input
	//     rejection is a backend validation the in-memory mock does not reproduce.
	t.Run("08_validation_rejects", func(t *testing.T) {
		// provider_side_validators: target_ref kind/identifier mismatches caught by the provider's own
		// validators (no backend involved), so both modes run them.
		t.Run("provider_side_validators", func(t *testing.T) {
			config, _, _, _ := testconfig.BBWorkspace(t)

			tenantWithName := config.WithFirstBlock(
				testconfig.Descend("spec", "target_ref")(testconfig.SetRawExpr(`{ kind = "meshTenant", name = "some-workspace" }`)),
			)
			workspaceWithUuid := config.WithFirstBlock(
				testconfig.Descend("spec", "target_ref")(testconfig.SetRawExpr(`{ kind = "meshWorkspace", uuid = "00000000-0000-0000-0000-000000000000" }`)),
			)

			ApplyAndTest(t, resource.TestCase{
				Steps: []resource.TestStep{
					{
						// target_ref kind=meshTenant must use uuid, not name.
						Config:      tenantWithName.String(),
						ExpectError: regexp.MustCompile(`must not be set when kind`),
					},
					{
						// target_ref kind=meshWorkspace must use name, not uuid.
						Config:      workspaceWithUuid.String(),
						ExpectError: regexp.MustCompile(`must not be set when kind`),
					},
				},
			})
		})

		// backend_rejections: rejections that only the real backend produces. The whole subtest is
		// skipped in mock — the in-memory mock does not reject a STATIC input assigned as a customer
		// input, so there is nothing here it could exercise.
		t.Run("backend_rejections", func(t *testing.T) {
			if IsMockClientTest() {
				t.Skip("backend-only validation (STATIC-input rejection) the mock does not reproduce")
			}
			config, buildingBlockAddr, _, _ := testconfig.BBWorkspace(t)

			// A STATIC BBD input (region) must not be accepted as a customer/operator input.
			invalidInputAssignment := config.WithFirstBlock(
				testconfig.Descend("spec", "inputs", "region")(testconfig.SetRawExpr(`{
  value = jsonencode("eu-central-1")
}`)),
			)

			ApplyAndTest(t, resource.TestCase{
				Steps: []resource.TestStep{
					{
						// Create a valid BB first, then (next step) attempt the invalid STATIC-input
						// assignment as an Update.
						Config: config.String(),
						ConfigPlanChecks: resource.ConfigPlanChecks{
							PreApply: []plancheck.PlanCheck{
								plancheck.ExpectResourceAction(buildingBlockAddr.String(), plancheck.ResourceActionCreate),
							},
						},
						ConfigStateChecks: []statecheck.StateCheck{
							statecheck.ExpectKnownValue(buildingBlockAddr.String(), tfjsonpath.New("metadata").AtMapKey("uuid"), xknownvalue.NotEmptyString()),
						},
					},
					{
						// Assigning the STATIC input as a customer input must be rejected.
						Config:      invalidInputAssignment.String(),
						ExpectError: regexp.MustCompile("is not defined as a customer or platform-operator input"),
					},
				},
			})
		})
	})

	// 09_purge_on_delete sets purge_on_delete = true so teardown deletes the BB via DELETE /{uuid}/purge.
	// Runs in both modes (create + flag wiring + purge teardown succeeds); the CheckDestroy below carries
	// the mechanism and its acceptance-only 404 assertion.
	t.Run("09_purge_on_delete", func(t *testing.T) {
		workspaceConfig, workspaceAddr := testconfig.Workspace(t)
		var buildingBlockDefinitionAddr testconfig.Traversal
		buildingBlockDefinitionConfig := testconfig.Resource{Name: "building_block", Suffix: "_01_workspace"}.TestSupportConfig(t, "").WithFirstBlock(
			testconfig.ExtractAddress(&buildingBlockDefinitionAddr),
			testconfig.OwnedByWorkspace(workspaceAddr),
		)

		var buildingBlockAddr testconfig.Traversal
		config := testconfig.Resource{Name: "building_block", Suffix: "_01_workspace"}.Config(t).WithFirstBlock(
			testconfig.ExtractAddress(&buildingBlockAddr),
			testconfig.Descend("spec", "building_block_definition_version_ref")(testconfig.SetRawExpr(`{ uuid = %s }`, buildingBlockDefinitionAddr.Join("version_latest", "uuid"))),
			testconfig.Descend("spec", "target_ref")(testconfig.SetAddr(workspaceAddr, "ref")),
			testconfig.Descend("purge_on_delete")(testconfig.SetRawExpr("true")),
		).Join(workspaceConfig, buildingBlockDefinitionConfig)

		var bbUuid string
		ApplyAndTest(t, resource.TestCase{
			// Runs after teardown destroyed the block (purged → soft-deleted) and its definition (whose
			// deletion hard-removes the soft-deleted block), so the block is gone and the GET 404s (Read → nil).
			// Acceptance-only: in mock mode the block is hard-removed from the store and the framework's own
			// destroy check covers it.
			CheckDestroy: func(*terraform.State) error {
				if IsMockClientTest() {
					return nil
				}
				bb, err := acceptanceClient(t).BuildingBlockV2.Read(context.Background(), bbUuid)
				if err != nil {
					return fmt.Errorf("reading purged building block %s: %w", bbUuid, err)
				}
				if bb != nil {
					return fmt.Errorf("expected building block %s to be gone after purge + definition teardown, but the GET still returned it", bbUuid)
				}
				return nil
			},
			Steps: []resource.TestStep{
				{
					Config: config.String(),
					ConfigPlanChecks: resource.ConfigPlanChecks{
						PreApply: []plancheck.PlanCheck{
							plancheck.ExpectResourceAction(buildingBlockAddr.String(), plancheck.ResourceActionCreate),
						},
					},
					ConfigStateChecks: []statecheck.StateCheck{
						statecheck.ExpectKnownValue(buildingBlockAddr.String(), tfjsonpath.New("metadata").AtMapKey("uuid"), xknownvalue.NotEmptyString(func(v string) error {
							bbUuid = v
							return nil
						})),
						statecheck.ExpectKnownValue(buildingBlockAddr.String(), tfjsonpath.New("purge_on_delete"), knownvalue.Bool(true)),
					},
				},
			},
		})
	})

	// 10_partial_input_ownership proves a configuration may manage only the inputs it declares: an input
	// it omits is preserved server-side and surfaced read-only in all_inputs, not dropped and not drift.
	// This is what lets a platform operator manage only operator inputs while the consumer owns the user
	// inputs (and vice versa). Runs in both modes — the mock preserves inputs omitted from a PUT just like
	// the backend, and the provider's Read drops un-declared inputs to all_inputs in both.
	t.Run("10_partial_input_ownership", func(t *testing.T) {
		workspaceConfig, workspaceAddr := testconfig.Workspace(t)

		// BBD whose `size` is a PLATFORM_OPERATOR_MANUAL_INPUT (name/environment are user inputs).
		var buildingBlockDefinitionAddr testconfig.Traversal
		buildingBlockDefinitionConfig := testconfig.Resource{Name: "building_block", Suffix: "_03_operator_inputs"}.TestSupportConfig(t, "").WithFirstBlock(
			testconfig.ExtractAddress(&buildingBlockDefinitionAddr),
			testconfig.OwnedByWorkspace(workspaceAddr),
		)

		// Create with all inputs declared (admin sets the operator input on create).
		var buildingBlockAddr testconfig.Traversal
		fullConfig := testconfig.Resource{Name: "building_block", Suffix: "_01_workspace"}.Config(t).WithFirstBlock(
			testconfig.ExtractAddress(&buildingBlockAddr),
			testconfig.Descend("spec", "building_block_definition_version_ref")(testconfig.SetAddr(buildingBlockDefinitionAddr, "version_latest")),
			testconfig.Descend("spec", "target_ref")(testconfig.SetAddr(workspaceAddr, "ref")),
		).Join(workspaceConfig, buildingBlockDefinitionConfig)

		// Then manage ONLY the operator input — drop the user inputs from the configuration.
		sizeOnlyConfig := fullConfig.WithFirstBlock(
			testconfig.Descend("spec", "inputs")(testconfig.SetRawExpr(`{ size = { value = jsonencode(16) } }`)),
		)

		ApplyAndTest(t, resource.TestCase{
			Steps: []resource.TestStep{
				{
					Config:            fullConfig.String(),
					ConfigStateChecks: bbv3StateChecks(buildingBlockAddr, "my-workspace-building-block", bbv3SizeEnvInputChecks(buildingBlockAddr)...),
				},
				{
					// Dropping the user inputs is an in-place Update, never a Replace.
					Config: sizeOnlyConfig.String(),
					ConfigPlanChecks: resource.ConfigPlanChecks{
						PreApply: []plancheck.PlanCheck{
							plancheck.ExpectResourceAction(buildingBlockAddr.String(), plancheck.ResourceActionUpdate),
						},
					},
					ConfigStateChecks: []statecheck.StateCheck{
						// The operator input stays in spec.inputs (it is declared).
						statecheck.ExpectKnownValue(buildingBlockAddr.String(), tfjsonpath.New("spec").AtMapKey("inputs").AtMapKey("size").AtMapKey("value"), knownvalue.StringExact("16")),
						// The un-declared user inputs are preserved, surfaced read-only in all_inputs.
						statecheck.ExpectKnownValue(buildingBlockAddr.String(), tfjsonpath.New("all_inputs").AtMapKey("name").AtMapKey("value"), knownvalue.StringExact(`"my-name"`)),
						statecheck.ExpectKnownValue(buildingBlockAddr.String(), tfjsonpath.New("all_inputs").AtMapKey("environment").AtMapKey("value"), knownvalue.StringExact(`"dev"`)),
						statecheck.ExpectKnownValue(buildingBlockAddr.String(), tfjsonpath.New("all_inputs").AtMapKey("size").AtMapKey("value"), knownvalue.StringExact("16")),
					},
				},
				{
					// No drift: the omitted user inputs must not reappear as a pending change.
					Config:   sizeOnlyConfig.String(),
					PlanOnly: true,
				},
			},
		})
	})

	t.Run("11_run_transparency_failed_run", func(t *testing.T) {
		if IsMockClientTest() {
			t.Skip("a failing terraform run and run-log transparency require the real backend")
		}

		// runCase walks an unprivileged, workspace-scoped API key (BUILDINGBLOCK_* only — no MANAGED_/ADM_
		// authority) through creating a cross-workspace building block from a terraform BBD pinned to the
		// `broken` ref, whose `tofu apply` fails on a deliberately-false precondition. The run fails either
		// way; whether the failing step's log is surfaced in the apply error is gated solely by the BBD's
		// run_transparency. Each case is fully isolated (its own workspaces, BBD, key) so the two opposite
		// outcomes cannot interfere.
		runCase := func(t *testing.T, runTransparency bool, expectErr *regexp.Regexp) {
			t.Helper()
			// Workspace W owns the BBD; workspace O consumes it across the workspace boundary.
			workspaceConfig, workspaceAddr := testconfig.Workspace(t)

			bbdResource := testconfig.Resource{Name: "building_block", Suffix: "_04_sensitive_user_input"}
			var bbdAddr testconfig.Traversal
			bbdConfig := bbdResource.TestSupportConfig(t, "_bbd").WithFirstBlock(
				testconfig.ExtractAddress(&bbdAddr),
				testconfig.OwnedByWorkspace(workspaceAddr),
				// Point the terraform implementation at the committed bare repo's `broken` branch, whose
				// module fails `tofu apply` on a precondition. In mock mode this whole test is skipped.
				testconfig.Descend("version_spec", "implementation", "terraform", "repository_url")(testconfig.SetRawExpr("%q", terraformTestdataRepoURL(t))),
				testconfig.Descend("version_spec", "implementation", "terraform", "ref_name")(testconfig.SetRawExpr("%q", "broken")),
				testconfig.Descend("spec", "run_transparency")(testconfig.SetRawExpr("%t", runTransparency)),
			)

			var otherWorkspaceAddr testconfig.Traversal
			otherWorkspaceConfig, _ := testconfig.Workspace(t)
			otherWorkspaceConfig = otherWorkspaceConfig.WithFirstBlock(
				testconfig.RenameKey("other"),
				testconfig.ExtractAddress(&otherWorkspaceAddr),
			)

			// Workspace-scoped key in the consumer workspace: BUILDINGBLOCK_* only, deliberately no
			// MANAGED_/ADM_ authority. Reading the failed run's logs must hinge purely on run transparency.
			apiKeyConfig, apiKeyAddr := testconfig.ApiKey(t, otherWorkspaceAddr)
			apiKeyConfig = apiKeyConfig.WithFirstBlock(
				testconfig.Descend("spec", "permissions")(testconfig.SetRawExpr(`["BUILDINGBLOCK_SAVE", "BUILDINGBLOCK_LIST", "BUILDINGBLOCK_DELETE"]`)),
			)

			step1Config := workspaceConfig.Join(bbdConfig, otherWorkspaceConfig, apiKeyConfig)

			// The consumer creates the BB via the meshstack-other provider (its workspace-scoped key).
			otherProviderConfig := testconfig.Resource{Name: "building_block"}.TestSupportConfig(t, "_other_provider")

			var buildingBlockAddr testconfig.Traversal
			bbConfig := bbdResource.TestSupportConfig(t, "").WithFirstBlock(
				testconfig.ExtractAddress(&buildingBlockAddr),
				testconfig.Descend("spec", "building_block_definition_version_ref")(testconfig.SetRawExpr(`{ uuid = %s }`, bbdAddr.Join("version_latest_release", "uuid"))),
				testconfig.Descend("spec", "target_ref")(testconfig.SetAddr(otherWorkspaceAddr, "ref")),
				testconfig.Descend("provider")(testconfig.SetRawExpr("meshstack-other")),
				// Keep the key alive until after the BB is destroyed: the other provider needs it for teardown.
				testconfig.Descend("depends_on")(testconfig.SetRawExpr("[%s]", apiKeyAddr)),
			)
			step2Config := bbConfig.Join(step1Config, otherProviderConfig)

			var apiKeyClientId, apiKeyClientSecret lazyVariable
			ApplyAndTest(t, resource.TestCase{
				Steps: []resource.TestStep{
					{
						// Admin mints the infra + the workspace-scoped key; capture its credentials.
						Config: step1Config.String(),
						ConfigStateChecks: []statecheck.StateCheck{
							statecheck.ExpectKnownValue(apiKeyAddr.String(), tfjsonpath.New("status").AtMapKey("client_id"), xknownvalue.NotEmptyString(func(clientId string) error {
								apiKeyClientId = lazyVariable(clientId)
								return nil
							})),
							statecheck.ExpectKnownValue(apiKeyAddr.String(), tfjsonpath.New("status").AtMapKey("client_secret"), xknownvalue.NotEmptyString(func(clientSecret string) error {
								apiKeyClientSecret = lazyVariable(clientSecret)
								return nil
							})),
						},
					},
					{
						// The consumer creates the BB; its run fails on the broken ref. wait_for_completion is
						// set in the example, so the apply errors on the failed run.
						Config: step2Config.String(),
						ConfigVariables: tfconfig.Variables{
							"apikey_client_id":     &apiKeyClientId,
							"apikey_client_secret": &apiKeyClientSecret,
						},
						ExpectError: expectErr,
					},
				},
			})
		}

		// Run transparency ON: the unprivileged workspace key may read the failed run, so the broken
		// module's precondition message is surfaced as part of the apply error.
		t.Run("transparency_on_surfaces_log", func(t *testing.T) {
			runCase(t, true, regexp.MustCompile("intentionally broken BBD version"))
		})
		// Run transparency OFF: the failed run is opaque to the workspace key (null run uuid), so the apply
		// errors with only the generic run failure — the step log is not leaked.
		t.Run("transparency_off_hides_log", func(t *testing.T) {
			runCase(t, false, regexp.MustCompile("Building block run failed"))
		})
	})

	// 12_moved_from_v2_with_secret guards the v2→v3 migration of a block with a sensitive USER_INPUT: the
	// move must plan in-place AND preserve the secret, even though secret_value is write-only and cannot
	// ride through state (moveFromV2 + secret.ValueToConverter handle the seed/refresh/echo). The v3
	// config re-declares the input with a DISTINCT placeholder and no secret_version; corruption is
	// detectable because sending the placeholder plaintext would change the all_inputs hash. Runs in both
	// modes (the mock's backendSecretBehavior mirrors the backend).
	t.Run("12_moved_from_v2_with_secret", func(t *testing.T) {
		workspaceConfig, workspaceAddr := testconfig.Workspace(t)

		// Shared sensitive BBD (api_key STRING + script CODE USER_INPUTs, static_secret STATIC), pointed
		// at the committed bare repo so acceptance runs execute quickly (unused in mock mode).
		var bbdAddr testconfig.Traversal
		bbdConfig := testconfig.Resource{Name: "building_block", Suffix: "_04_sensitive_user_input"}.TestSupportConfig(t, "_bbd").WithFirstBlock(
			testconfig.ExtractAddress(&bbdAddr),
			testconfig.OwnedByWorkspace(workspaceAddr),
			testconfig.Descend("version_spec", "implementation", "terraform", "repository_url")(testconfig.SetRawExpr("%q", terraformTestdataRepoURL(t))),
		)

		// v2 block supplying the real secrets via value_string_sensitive/value_code_sensitive.
		var v2Addr testconfig.Traversal
		v2Config := testconfig.Resource{Name: "building_block_v2", Suffix: "_05_moved_secret"}.TestSupportConfig(t, "").WithFirstBlock(
			testconfig.ExtractAddress(&v2Addr),
			testconfig.Descend("spec", "building_block_definition_version_ref")(testconfig.SetAddr(bbdAddr, "version_latest")),
			testconfig.Descend("spec", "target_ref")(testconfig.SetAddr(workspaceAddr, "ref")),
		).Join(workspaceConfig, bbdConfig)

		// v3 block: same definition + target; re-declares the secrets with DISTINCT placeholders, no version.
		var v3Addr testconfig.Traversal
		buildV3 := func() testconfig.Config {
			return testconfig.Resource{Name: "building_block", Suffix: "_05_moved_secret"}.TestSupportConfig(t, "").WithFirstBlock(
				testconfig.ExtractAddress(&v3Addr),
				testconfig.Descend("spec", "building_block_definition_version_ref")(testconfig.SetAddr(bbdAddr, "version_latest")),
				testconfig.Descend("spec", "target_ref")(testconfig.SetAddr(workspaceAddr, "ref")),
			).Join(workspaceConfig, bbdConfig)
		}
		v3Config := buildV3()
		// Rotation: bump secret_version (null→"2") + new value → the secret is re-applied and the hash changes.
		v3Rotated := buildV3().WithFirstBlock(
			testconfig.Descend("spec", "inputs", "api_key", "sensitive", "secret_value")(testconfig.SetRawExpr(`"rotated-real-api-key"`)),
			testconfig.Descend("spec", "inputs", "api_key", "sensitive", "secret_version")(testconfig.SetRawExpr(`"2"`)),
		)

		movedConfig := testconfig.Resource{Name: "building_block"}.TestSupportConfig(t, "_moved_from_v2_secret")

		// Capture the v2 block's secret hashes so the post-move step can assert they are preserved, not
		// overwritten. Both a STRING (api_key, surfaced in combined_inputs.api_key.value_string) and a
		// CODE (script, surfaced in combined_inputs.script.value_code) sensitive input are covered — the
		// two take the identical code path and both work in mock and acc (the mock's backendSecretBehavior
		// hashes the SecretEmbedded plaintext once the outbound DTO carries IsSensitive=true).
		var v2ApiKeyHash, v2ScriptHash string
		captureApiKeyHash := xknownvalue.NotEmptyString(func(v string) error { v2ApiKeyHash = v; return nil })
		captureScriptHash := xknownvalue.NotEmptyString(func(v string) error { v2ScriptHash = v; return nil })

		ApplyAndTest(t, resource.TestCase{
			Steps: []resource.TestStep{
				{
					Config: v2Config.String(),
					ConfigPlanChecks: resource.ConfigPlanChecks{
						PreApply: []plancheck.PlanCheck{plancheck.ExpectResourceAction(v2Addr.String(), plancheck.ResourceActionCreate)},
					},
					ConfigStateChecks: []statecheck.StateCheck{
						statecheck.ExpectKnownValue(v2Addr.String(), tfjsonpath.New("metadata").AtMapKey("uuid"), xknownvalue.NotEmptyString()),
						statecheck.ExpectKnownValue(v2Addr.String(), tfjsonpath.New("spec").AtMapKey("combined_inputs").AtMapKey("api_key").AtMapKey("value_string"), captureApiKeyHash),
						statecheck.ExpectKnownValue(v2Addr.String(), tfjsonpath.New("spec").AtMapKey("combined_inputs").AtMapKey("script").AtMapKey("value_code"), captureScriptHash),
					},
				},
				{
					// The move must plan as an in-place Update (never Replace), and the secrets must be
					// preserved: the v3 all_inputs hashes must equal the v2 hashes despite the placeholders.
					Config: v3Config.Join(movedConfig).String(),
					ConfigPlanChecks: resource.ConfigPlanChecks{
						PreApply: []plancheck.PlanCheck{plancheck.ExpectResourceAction(v3Addr.String(), plancheck.ResourceActionUpdate)},
					},
					ConfigStateChecks: []statecheck.StateCheck{
						statecheck.ExpectKnownValue(v3Addr.String(),
							tfjsonpath.New("all_inputs").AtMapKey("api_key").AtMapKey("sensitive").AtMapKey("secret_hash"),
							xknownvalue.NotEmptyString(func(v string) error {
								if v != v2ApiKeyHash {
									return fmt.Errorf("api_key secret was not preserved across the move: hash %q != v2 hash %q (the re-supplied placeholder overwrote the secret)", v, v2ApiKeyHash)
								}
								return nil
							})),
						statecheck.ExpectKnownValue(v3Addr.String(),
							tfjsonpath.New("all_inputs").AtMapKey("script").AtMapKey("sensitive").AtMapKey("secret_hash"),
							xknownvalue.NotEmptyString(func(v string) error {
								if v != v2ScriptHash {
									return fmt.Errorf("script secret was not preserved across the move: hash %q != v2 hash %q (the re-supplied placeholder overwrote the secret)", v, v2ScriptHash)
								}
								return nil
							})),
					},
				},
				{
					// Rotating api_key (new value + bumped secret_version) re-applies the secret: its hash changes.
					Config: v3Rotated.Join(movedConfig).String(),
					ConfigStateChecks: []statecheck.StateCheck{
						statecheck.ExpectKnownValue(v3Addr.String(),
							tfjsonpath.New("all_inputs").AtMapKey("api_key").AtMapKey("sensitive").AtMapKey("secret_hash"),
							xknownvalue.NotEmptyString(func(v string) error {
								if v == v2ApiKeyHash {
									return fmt.Errorf("api_key hash unchanged after rotation: still %q", v)
								}
								return nil
							})),
					},
				},
			},
		})
	})

}

// bbv3StateChecks returns the baseline state checks shared by every BB v3 create/move step: a
// non-empty metadata.uuid, the display name, the always-present "name" input, a non-empty status, and
// a non-empty latest_run_uuid. Callers append scenario-specific input checks via extra — the
// workspace and tenant examples both pass bbv3SizeEnvInputChecks.
func bbv3StateChecks(buildingBlockAddr testconfig.Traversal, displayName string, extra ...statecheck.StateCheck) []statecheck.StateCheck {
	checks := []statecheck.StateCheck{
		statecheck.ExpectKnownValue(buildingBlockAddr.String(), tfjsonpath.New("metadata").AtMapKey("uuid"), xknownvalue.NotEmptyString()),
		statecheck.ExpectKnownValue(buildingBlockAddr.String(), tfjsonpath.New("spec").AtMapKey("display_name"), knownvalue.StringExact(displayName)),
		statecheck.ExpectKnownValue(buildingBlockAddr.String(), tfjsonpath.New("spec").AtMapKey("inputs").AtMapKey("name").AtMapKey("value"), knownvalue.StringExact(`"my-name"`)),
		statecheck.ExpectKnownValue(buildingBlockAddr.String(), tfjsonpath.New("status").AtMapKey("status"), xknownvalue.NotEmptyString()),
		statecheck.ExpectKnownValue(buildingBlockAddr.String(), tfjsonpath.New("status").AtMapKey("latest_run_uuid"), xknownvalue.NotEmptyString()),
	}
	return append(checks, extra...)
}

// bbv3SizeEnvInputChecks are the size/environment input-value checks shared by the workspace and
// tenant examples (resource_01_workspace.tf / resource_02_tenant.tf) and the moved-from-v2 scenario.
func bbv3SizeEnvInputChecks(buildingBlockAddr testconfig.Traversal) []statecheck.StateCheck {
	return []statecheck.StateCheck{
		statecheck.ExpectKnownValue(buildingBlockAddr.String(), tfjsonpath.New("spec").AtMapKey("inputs").AtMapKey("size").AtMapKey("value"), knownvalue.StringExact("16")),
		statecheck.ExpectKnownValue(buildingBlockAddr.String(), tfjsonpath.New("spec").AtMapKey("inputs").AtMapKey("environment").AtMapKey("value"), knownvalue.StringExact(`"dev"`)),
	}
}

// Test_compareContentHashes tests the content hash comparison logic for the building block rerun decision.
func Test_compareContentHashes(t *testing.T) {
	v2a := BuildingBlockDefinitionVersionContentHash{hashVersion: 2, hashValue: "aaa"}.toBase64()
	v2b := BuildingBlockDefinitionVersionContentHash{hashVersion: 2, hashValue: "bbb"}.toBase64()
	v1 := "v1:someLegacyHashValue"

	tests := []struct {
		name      string
		planHash  string
		stateHash string
		want      hashComparison
	}{
		{"versioned, same version, same value", v2a, v2a, hashSame},
		{"versioned, same version, different value", v2a, v2b, hashDifferent},
		{"versioned, different algorithm version", v2a, v1, hashIncomparable},
		{"different algorithm version, other direction", v1, v2a, hashIncomparable},
		{"free-form, changed", "2", "1", hashDifferent},
		{"free-form, unchanged", "x", "x", hashSame},
		{"free-form plan vs versioned state", "manual", v2a, hashDifferent},
		{"versioned plan vs free-form state", v2a, "manual", hashDifferent},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.Equal(t, tt.want, compareContentHashes(tt.planHash, tt.stateHash))
		})
	}
}

func Test_rerunNeeded(t *testing.T) {
	v2a := BuildingBlockDefinitionVersionContentHash{hashVersion: 2, hashValue: "aaa"}.toBase64()
	v2b := BuildingBlockDefinitionVersionContentHash{hashVersion: 2, hashValue: "bbb"}.toBase64()
	v1 := "v1:someLegacyHashValue"

	spec := func(uuid string, contentHash *string, inputs map[string]*client.MeshBuildingBlockInput, parents ...client.MeshBuildingBlockParent) client.MeshBuildingBlockV2Spec {
		return client.MeshBuildingBlockV2Spec{
			BuildingBlockDefinitionVersionRef: client.MeshBuildingBlockV2DefinitionVersionRef{
				Uuid:        uuid,
				ContentHash: contentHash,
			},
			Inputs:               inputs,
			ParentBuildingBlocks: parents,
		}
	}
	input := func(sensitive bool) *client.MeshBuildingBlockInput {
		return &client.MeshBuildingBlockInput{IsSensitive: sensitive}
	}
	parent := client.MeshBuildingBlockParent{BuildingBlockUuid: "parent-1", DefinitionUuid: "def-1"}

	tests := []struct {
		name  string
		plan  client.MeshBuildingBlockV2Spec
		state client.MeshBuildingBlockV2Spec
		want  bool
	}{
		{"uuid differs", spec("uuid-2", nil, nil), spec("uuid-1", nil, nil), true},
		{"content_hash newly set", spec("uuid", &v2a, nil), spec("uuid", nil, nil), true},
		{"content_hash removed", spec("uuid", nil, nil), spec("uuid", &v2a, nil), false},
		{"content_hash version mismatch", spec("uuid", &v2a, nil), spec("uuid", &v1, nil), false},
		{"content_hash changed", spec("uuid", &v2a, nil), spec("uuid", &v2b, nil), true},
		{"content_hash arbitrary value changed", spec("uuid", new("force-rerun"), nil), spec("uuid", new("previous-value"), nil), true},
		{"content_hash unchanged", spec("uuid", &v2a, nil), spec("uuid", &v2a, nil), false},
		{"inputs added", spec("uuid", nil, map[string]*client.MeshBuildingBlockInput{"a": input(false)}), spec("uuid", nil, nil), true},
		{"inputs changed", spec("uuid", nil, map[string]*client.MeshBuildingBlockInput{"a": input(true)}), spec("uuid", nil, map[string]*client.MeshBuildingBlockInput{"a": input(false)}), true},
		{"inputs unchanged", spec("uuid", nil, map[string]*client.MeshBuildingBlockInput{"a": input(false)}), spec("uuid", nil, map[string]*client.MeshBuildingBlockInput{"a": input(false)}), false},
		{"parents differ", spec("uuid", nil, nil, parent), spec("uuid", nil, nil), true},
		{"parents unchanged", spec("uuid", nil, nil, parent), spec("uuid", nil, nil, parent), false},
		{"all equal", spec("uuid", &v2a, map[string]*client.MeshBuildingBlockInput{"a": input(false)}, parent), spec("uuid", &v2a, map[string]*client.MeshBuildingBlockInput{"a": input(false)}, parent), false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.Equal(t, tt.want, rerunNeeded(tt.plan, tt.state))
		})
	}
}
