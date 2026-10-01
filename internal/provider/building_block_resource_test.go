package provider

import (
	"context"
	"fmt"
	"regexp"
	"testing"
	"time"

	"github.com/hashicorp/terraform-plugin-testing/compare"
	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/knownvalue"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
	"github.com/hashicorp/terraform-plugin-testing/statecheck"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	"github.com/hashicorp/terraform-plugin-testing/tfjsonpath"
	"github.com/meshcloud/meshstack-cli/client"
	"github.com/meshcloud/meshstack-cli/pkg/auth"
	"github.com/stretchr/testify/require"

	"github.com/meshcloud/terraform-provider-meshstack/examples"
	"github.com/meshcloud/terraform-provider-meshstack/internal/provider/acctest/xknownvalue"
)

// Addresses of the remaining blocks in examples/resources/meshstack_building_block/*-test-*.tf, on
// top of the ones building_block_data_source_test.go declares. Every definition test-support file
// here declares the same `example` label, so which file a config composes decides which definition
// the step files wire themselves to.
const (
	buildingBlockSensitiveAddr   = "meshstack_building_block.sensitive_user_input"
	buildingBlockMovedSecretAddr = "meshstack_building_block.moved_secret"
	bbv2MovedSecretAddr          = "meshstack_building_block_v2.moved_secret"
	bbBbdAddr                    = "meshstack_building_block_definition.example"
	bbSensitiveBbdAddr           = "meshstack_building_block_definition.sensitive_user_input"
)

// bbWorkspaceStepConfig is a workspace building block step with the definition it instantiates and
// the workspace both live in. The support names pick that definition, and anything else the step
// file references.
func bbWorkspaceStepConfig(t *testing.T, index int, supports ...string) string {
	t.Helper()
	return examples.JoinTestStepConfigs(
		examples.Resource.TestStepConfig(t, "building_block", index, append([]string{"lifecycle-variables"}, supports...)...),
		examples.Resource.TestStepConfig(t, "workspace", 1, "variables", "prerequisites"),
	)
}

// bbCrossWorkspaceSupportConfig is everything a cross-workspace flow needs before the block itself:
// the definition and the workspace owning it, the consumer workspace the block will live in, and the
// key authorizing the consumer side. apiKeyIndex picks what authority that key carries.
func bbCrossWorkspaceSupportConfig(t *testing.T, apiKeyIndex int, supports ...string) string {
	t.Helper()
	return examples.JoinTestStepConfigs(
		examples.Resource.TestSupportConfigs(t, "building_block", append([]string{"lifecycle-variables"}, supports...)...),
		examples.Resource.TestStepConfig(t, "api_key", apiKeyIndex),
		examples.Resource.TestStepConfig(t, "workspace", 1, "variables", "prerequisites", "consumer-workspace"),
	)
}

// bbCrossWorkspaceStepConfig adds the block itself, plus the meshstack-other provider alias that the
// key minted by the preceding step backs.
func bbCrossWorkspaceStepConfig(t *testing.T, index, apiKeyIndex int, supports ...string) string {
	t.Helper()
	return examples.JoinTestStepConfigs(
		examples.Resource.TestStepConfig(t, "building_block", index),
		bbCrossWorkspaceSupportConfig(t, apiKeyIndex, supports...),
		examples.Resource.TestSupportConfigs(t, "api_key", "other_provider"),
	)
}

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
	// The credential resolves the same way a real provider run resolves it from an empty provider
	// block: the MESHSTACK_* environment variables supply the whole credential, and nothing is read
	// from or written to a profile.
	c, err := auth.ResolveClient(context.Background(), auth.ResolveClientOptions{
		Version: "acctest", GitHubRepo: gitHubRepo,
	})
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
		vars := NewVariablesWithSuffix(acctest.RandString(8))
		// Renaming display_name must be an in-place Update and must not change anything else.
		renamedVars := With(vars, "bb_display_name", "my-workspace-building-block-renamed")
		updatedInputsVars := With(renamedVars, "bb_environment", "staging")
		// content_hash tracks the BBD version's content; setting it, then changing it, simulates the
		// BBD being updated and must trigger a rerun even though the version uuid is unchanged.
		contentHashV1Vars := With(updatedInputsVars, "bb_content_hash", "v1")
		contentHashV2Vars := With(updatedInputsVars, "bb_content_hash", "v2")

		config := bbWorkspaceStepConfig(t, 1, "01_workspace")
		// Step file 5 is step file 1 with the whole version object behind the ref plus an explicit
		// content_hash. Step file 6 adds a parent on top of that; the uuid there is synthetic, which is
		// why only a plan decision can be observed on it.
		contentHashConfig := bbWorkspaceStepConfig(t, 5, "01_workspace")
		withParentsConfig := bbWorkspaceStepConfig(t, 6, "01_workspace")

		ApplyAndTest(t, resource.TestCase{
			Steps: []resource.TestStep{
				{
					Config:          config,
					ConfigVariables: vars,
					ConfigPlanChecks: resource.ConfigPlanChecks{
						PreApply: []plancheck.PlanCheck{
							plancheck.ExpectResourceAction(buildingBlockWorkspaceAddr, plancheck.ResourceActionCreate),
						},
					},
					ConfigStateChecks: bbv3StateChecks(buildingBlockWorkspaceAddr, "my-workspace-building-block", bbv3SizeEnvInputChecks(buildingBlockWorkspaceAddr)...),
				},
				{
					// Import with verify. content_hash is json:"-" and never returned by the API;
					// wait_for_completion and purge_on_delete are config-only defaults — all excluded.
					ImportState:                          true,
					ImportStateVerify:                    true,
					ImportStateVerifyIdentifierAttribute: "metadata.uuid",
					ImportStateVerifyIgnore:              []string{"spec.building_block_definition_version_ref.content_hash", "wait_for_completion", "purge_on_delete", "timeouts.create", "timeouts.update", "timeouts.delete"},
					ImportStateIdFunc: func(s *terraform.State) (string, error) {
						rs := s.RootModule().Resources[buildingBlockWorkspaceAddr]
						if rs == nil {
							return "", fmt.Errorf("resource not found: %s", buildingBlockWorkspaceAddr)
						}
						return rs.Primary.Attributes["metadata.uuid"], nil
					},
					ResourceName:    buildingBlockWorkspaceAddr,
					ConfigVariables: vars,
				},
				{
					// The refreshed plan must be empty: unconfigured optional USER_INPUTs that the backend
					// echoes as null rows must not surface as drift. PlanOnly without ExpectNonEmptyPlan
					// asserts an empty plan. Runs in both modes — the mock materializes the same null rows,
					// so the check holds there too and we keep mock/acceptance behaviour in lock-step.
					Config:          config,
					ConfigVariables: vars,
					PlanOnly:        true,
				},
				{
					// Rename only display_name → in-place Update, never Replace.
					Config:          config,
					ConfigVariables: renamedVars,
					ConfigPlanChecks: resource.ConfigPlanChecks{
						PreApply: []plancheck.PlanCheck{
							plancheck.ExpectResourceAction(buildingBlockWorkspaceAddr, plancheck.ResourceActionUpdate),
						},
					},
					ConfigStateChecks: []statecheck.StateCheck{
						statecheck.ExpectKnownValue(buildingBlockWorkspaceAddr, tfjsonpath.New("spec").AtMapKey("display_name"), knownvalue.StringExact("my-workspace-building-block-renamed")),
					},
				},
				{
					// Change an input value → in-place Update.
					Config:          config,
					ConfigVariables: updatedInputsVars,
					ConfigPlanChecks: resource.ConfigPlanChecks{
						PreApply: []plancheck.PlanCheck{
							plancheck.ExpectResourceAction(buildingBlockWorkspaceAddr, plancheck.ResourceActionUpdate),
						},
					},
					ConfigStateChecks: []statecheck.StateCheck{
						statecheck.ExpectKnownValue(buildingBlockWorkspaceAddr, tfjsonpath.New("spec").AtMapKey("inputs").AtMapKey("environment").AtMapKey("value"), knownvalue.StringExact(`"staging"`)),
					},
				},
				{
					// Set initial content_hash to track BBD version "v1".
					Config:          contentHashConfig,
					ConfigVariables: contentHashV1Vars,
					ConfigPlanChecks: resource.ConfigPlanChecks{
						PreApply: []plancheck.PlanCheck{
							plancheck.ExpectResourceAction(buildingBlockWorkspaceAddr, plancheck.ResourceActionUpdate),
						},
					},
					ConfigStateChecks: []statecheck.StateCheck{
						statecheck.ExpectKnownValue(buildingBlockWorkspaceAddr, tfjsonpath.New("spec").AtMapKey("building_block_definition_version_ref").AtMapKey("content_hash"), knownvalue.StringExact("v1")),
					},
				},
				{
					// Bumping content_hash "v1"→"v2" simulates a BBD content update and triggers a rerun.
					Config:          contentHashConfig,
					ConfigVariables: contentHashV2Vars,
					ConfigPlanChecks: resource.ConfigPlanChecks{
						PreApply: []plancheck.PlanCheck{
							plancheck.ExpectResourceAction(buildingBlockWorkspaceAddr, plancheck.ResourceActionUpdate),
						},
					},
					ConfigStateChecks: []statecheck.StateCheck{
						statecheck.ExpectKnownValue(buildingBlockWorkspaceAddr, tfjsonpath.New("spec").AtMapKey("building_block_definition_version_ref").AtMapKey("content_hash"), knownvalue.StringExact("v2")),
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
					Config:          withParentsConfig,
					ConfigVariables: contentHashV2Vars,
					ConfigPlanChecks: resource.ConfigPlanChecks{
						PreApply: []plancheck.PlanCheck{
							plancheck.ExpectResourceAction(buildingBlockWorkspaceAddr, plancheck.ResourceActionDestroyBeforeCreate),
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
		vars := With(NewVariablesWithSuffix(acctest.RandString(8)), "terraform_repository_url", terraformTestdataRepoURL(t))
		config := examples.JoinTestStepConfigs(
			examples.Resource.TestStepConfig(t, "building_block", 4, "02_tenant", "variables"),
			tenantStepConfig(t, 1, 8, 1),
		)

		// api_key (sensitive STRING USER_INPUT) surfaces as a non-empty hash in all_inputs in both modes.
		sensitiveInputChecks := append(bbv3SizeEnvInputChecks(buildingBlockTenantAddr),
			statecheck.ExpectKnownValue(buildingBlockTenantAddr,
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
				statecheck.ExpectKnownValue(buildingBlockTenantAddr,
					tfjsonpath.New("status").AtMapKey("outputs").AtMapKey("api_key_echo").AtMapKey("value"),
					knownvalue.StringExact(`"super-secret-api-key"`)),
			)
		}

		ApplyAndTest(t, resource.TestCase{
			Steps: []resource.TestStep{
				{
					Config:          config,
					ConfigVariables: vars,
					ConfigPlanChecks: resource.ConfigPlanChecks{
						PreApply: []plancheck.PlanCheck{
							plancheck.ExpectResourceAction(buildingBlockTenantAddr, plancheck.ResourceActionCreate),
						},
					},
					ConfigStateChecks: bbv3StateChecks(buildingBlockTenantAddr, "my-tenant-building-block", sensitiveInputChecks...),
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
						rs := s.RootModule().Resources[buildingBlockTenantAddr]
						if rs == nil {
							return "", fmt.Errorf("resource not found: %s", buildingBlockTenantAddr)
						}
						return rs.Primary.Attributes["metadata.uuid"], nil
					},
					ResourceName:    buildingBlockTenantAddr,
					ConfigVariables: vars,
				},
			},
		})
	})

	// 03_workspace_moved_from_v2 guards the v2→v3 migration: a `moved` block from
	// meshstack_building_block_v2 to v3 must plan as an in-place Update, never a destroy+recreate.
	// moveFromV2 leaves target_ref/version_ref to be filled by the post-move refresh-Read, so the
	// RequiresReplace modifiers see equal values and do not fire.
	t.Run("03_workspace_moved_from_v2", func(t *testing.T) {
		vars := NewVariablesWithSuffix(acctest.RandString(8))
		// The v2 block and the v3 block it moves to run on the same definition, so both steps compose the
		// v2 example's definition.
		v2Config := examples.JoinTestStepConfigs(
			examples.Resource.TestStepConfig(t, "building_block_v2", 1, "01_workspace"),
			examples.Resource.TestStepConfig(t, "workspace", 1, "variables", "prerequisites"),
		)
		// Step file 7 wires the whole version object, which is what a configuration written against a
		// definition in the same state does.
		v3Config := examples.JoinTestStepConfigs(
			bbWorkspaceStepConfig(t, 7),
			examples.Resource.TestSupportConfigs(t, "building_block_v2", "01_workspace"),
		)
		// The moved-block source/target addresses are fixed (resource labels are not randomized), so the
		// test-support file hard-codes them directly.
		movedConfig := examples.Resource.TestSupportConfigs(t, "building_block", "moved_from_v2")

		ApplyAndTest(t, resource.TestCase{
			Steps: []resource.TestStep{
				{
					Config:          v2Config,
					ConfigVariables: vars,
					ConfigPlanChecks: resource.ConfigPlanChecks{
						PreApply: []plancheck.PlanCheck{
							plancheck.ExpectResourceAction(bbv2WorkspaceAddr, plancheck.ResourceActionCreate),
						},
					},
					ConfigStateChecks: []statecheck.StateCheck{
						statecheck.ExpectKnownValue(bbv2WorkspaceAddr, tfjsonpath.New("metadata").AtMapKey("uuid"), xknownvalue.NotEmptyString()),
					},
				},
				{
					// The move must plan as an in-place Update; a regression to Replace fails here.
					Config:          examples.JoinTestStepConfigs(v3Config, movedConfig),
					ConfigVariables: vars,
					ConfigPlanChecks: resource.ConfigPlanChecks{
						PreApply: []plancheck.PlanCheck{
							plancheck.ExpectResourceAction(buildingBlockWorkspaceAddr, plancheck.ResourceActionUpdate),
						},
					},
					ConfigStateChecks: bbv3StateChecks(buildingBlockWorkspaceAddr, "my-workspace-building-block", bbv3SizeEnvInputChecks(buildingBlockWorkspaceAddr)...),
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
		vars := NewVariablesWithSuffix(acctest.RandString(8))
		// Dedicated migration fixtures (manual impl, no sensitive inputs): the v1 legacy resource cannot
		// carry sensitive inputs, so this test stays decoupled from the terraform + sensitive _02_tenant
		// showcase, which both the v1 and v3 sides would otherwise have to satisfy.
		tenantConfig := examples.JoinTestStepConfigs(
			examples.Resource.TestSupportConfigs(t, "building_block", "tenant_migration_bbd"),
			tenantStepConfig(t, 1, 8, 1),
		)
		v1Config := examples.JoinTestStepConfigs(
			examples.Resource.TestStepConfig(t, "buildingblock", 2),
			tenantConfig,
		)
		v3Config := examples.JoinTestStepConfigs(
			examples.Resource.TestStepConfig(t, "building_block", 18),
			tenantConfig,
		)
		// The moved-block source/target addresses are fixed (resource labels are not randomized), so the
		// test-support file hard-codes them directly.
		movedConfig := examples.Resource.TestSupportConfigs(t, "building_block", "moved_from_v1")

		var v1Uuid string
		ApplyAndTest(t, resource.TestCase{
			Steps: []resource.TestStep{
				{
					Config:          v1Config,
					ConfigVariables: vars,
					ConfigPlanChecks: resource.ConfigPlanChecks{
						PreApply: []plancheck.PlanCheck{
							plancheck.ExpectResourceAction(buildingBlockV1ResourceAddr, plancheck.ResourceActionCreate),
						},
					},
					ConfigStateChecks: []statecheck.StateCheck{
						statecheck.ExpectKnownValue(buildingBlockV1ResourceAddr, tfjsonpath.New("metadata").AtMapKey("uuid"), xknownvalue.NotEmptyString(func(uuid string) error {
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
					Config:          examples.JoinTestStepConfigs(v3Config, movedConfig),
					ConfigVariables: vars,
					// Verified against a live backend (Plan: 0 add, 1 change, 0 destroy; metadata.uuid
					// preserved across the move). A regression to Replace fails here.
					ConfigPlanChecks: resource.ConfigPlanChecks{
						PreApply: []plancheck.PlanCheck{
							plancheck.ExpectResourceAction(buildingBlockTenantAddr, plancheck.ResourceActionUpdate),
						},
					},
					ConfigStateChecks: bbv3StateChecks(buildingBlockTenantAddr, "my-tenant-building-block"),
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
		// Workspace A owns the BBD (which declares `size` as PLATFORM_OPERATOR_MANUAL_INPUT) and the API
		// key used to set the operator input. Workspace B is the consumer: the building block lives there,
		// across the workspace boundary from the definition owner.
		vars := NewVariablesWithSuffix(acctest.RandString(8))
		var apiKeyClientId, apiKeyClientSecret lazyVariable
		keyVars := With(vars, "apikey_client_id", &apiKeyClientId, "apikey_client_secret", &apiKeyClientSecret)
		renamedVars := With(keyVars, "bb_name", "updated-name")
		// v2 of the definition adds a defaulted platform-operator input `tier`. Re-draft (new v2 draft;
		// version_latest_release still v1) then re-release (v2 released) mirror the version dance in 06, so
		// the upgrade steps below prove the backend applies an operator-input default on upgrade.
		redraftVars := With(renamedVars, "bbd_draft", true)

		// Step 1 sets up the infrastructure and mints the key (default/admin provider). API key 7 is the
		// MANAGED_BUILDINGBLOCK_SAVE key whose cross-workspace authority is the capability under test.
		step1Config := bbCrossWorkspaceSupportConfig(t, 7, "03_operator_inputs")

		// Steps 2+ keep the meshstack-other provider alias configured (the MANAGED key minted above); the
		// block itself is admin-managed, so every step passes the key credentials as config variables. Step
		// file 13 leaves the operator input out, 14 supplies it; the -tier definition edits the same
		// resource, so swapping it in is that definition's v2.
		createConfig := bbCrossWorkspaceStepConfig(t, 13, 7, "03_operator_inputs")
		suppliedConfig := bbCrossWorkspaceStepConfig(t, 14, 7, "03_operator_inputs")
		upgradeConfig := bbCrossWorkspaceStepConfig(t, 14, 7, "03_operator_inputs-tier", "bbd-variables")

		// On a real backend the operator-input-less create parks WAITING_FOR_OPERATOR_INPUT; the mock has
		// no such state and short-circuits the create to a terminal status.
		createStatusCheck := statecheck.ExpectKnownValue(buildingBlockWorkspaceAddr, tfjsonpath.New("status").AtMapKey("status"), knownvalue.StringExact("WAITING_FOR_OPERATOR_INPUT"))
		if IsMockClientTest() {
			createStatusCheck = statecheck.ExpectKnownValue(buildingBlockWorkspaceAddr, tfjsonpath.New("status").AtMapKey("status"), knownvalue.StringExact("SUCCEEDED"))
		}

		// Post-upgrade checks: the block reaches SUCCEEDED in both modes. The `tier` assertion is
		// acceptance-only — applying a defaulted operator input on upgrade is backend behaviour the
		// mock does not reproduce (it neither runs the block nor materializes the added default).
		upgradedChecks := []statecheck.StateCheck{
			statecheck.ExpectKnownValue(buildingBlockWorkspaceAddr, tfjsonpath.New("status").AtMapKey("status"), knownvalue.StringExact("SUCCEEDED")),
		}
		if !IsMockClientTest() {
			upgradedChecks = append(upgradedChecks,
				statecheck.ExpectKnownValue(buildingBlockWorkspaceAddr, tfjsonpath.New("all_inputs").AtMapKey("tier").AtMapKey("value"), knownvalue.StringExact(`"bronze"`)),
			)
		}

		ApplyAndTest(t, resource.TestCase{
			Steps: []resource.TestStep{
				{
					// Mint the MANAGED key (default/admin provider) and capture its credentials.
					Config:          step1Config,
					ConfigVariables: vars,
					ConfigStateChecks: []statecheck.StateCheck{
						statecheck.ExpectKnownValue(apiKeyResourceAddr, tfjsonpath.New("status").AtMapKey("client_id"), xknownvalue.NotEmptyString(func(clientId string) error {
							apiKeyClientId = lazyVariable(clientId)
							return nil
						})),
						statecheck.ExpectKnownValue(apiKeyResourceAddr, tfjsonpath.New("status").AtMapKey("client_secret"), xknownvalue.NotEmptyString(func(clientSecret string) error {
							apiKeyClientSecret = lazyVariable(clientSecret)
							return nil
						})),
					},
				},
				{
					// Create without the operator input → on a real backend the block parks
					// WAITING_FOR_OPERATOR_INPUT (the provider surfaces a warning, not an error).
					Config:          createConfig,
					ConfigVariables: keyVars,
					ConfigPlanChecks: resource.ConfigPlanChecks{
						PreApply: []plancheck.PlanCheck{
							plancheck.ExpectResourceAction(buildingBlockWorkspaceAddr, plancheck.ResourceActionCreate),
						},
					},
					ConfigStateChecks: []statecheck.StateCheck{
						statecheck.ExpectKnownValue(buildingBlockWorkspaceAddr, tfjsonpath.New("spec").AtMapKey("inputs").AtMapKey("name").AtMapKey("value"), knownvalue.StringExact(`"my-name"`)),
						createStatusCheck,
					},
				},
				{
					// Supplying `size` via PUT resumes provisioning; the provider must poll THROUGH the
					// transient WAITING to the resulting SUCCEEDED run instead of returning on the stale WAITING.
					Config:          suppliedConfig,
					ConfigVariables: keyVars,
					ConfigPlanChecks: resource.ConfigPlanChecks{
						PreApply: []plancheck.PlanCheck{
							plancheck.ExpectResourceAction(buildingBlockWorkspaceAddr, plancheck.ResourceActionUpdate),
						},
					},
					ConfigStateChecks: []statecheck.StateCheck{
						statecheck.ExpectKnownValue(buildingBlockWorkspaceAddr, tfjsonpath.New("all_inputs").AtMapKey("size").AtMapKey("value"), knownvalue.StringExact("16")),
						statecheck.ExpectKnownValue(buildingBlockWorkspaceAddr, tfjsonpath.New("status").AtMapKey("status"), knownvalue.StringExact("SUCCEEDED")),
						statecheck.ExpectKnownValue(buildingBlockWorkspaceAddr, tfjsonpath.New("status").AtMapKey("latest_run_uuid"), xknownvalue.NotEmptyString()),
					},
				},
				{
					// Changing a consumer input is an in-place Update; the operator input stays put.
					Config:          suppliedConfig,
					ConfigVariables: renamedVars,
					ConfigPlanChecks: resource.ConfigPlanChecks{
						PreApply: []plancheck.PlanCheck{
							plancheck.ExpectResourceAction(buildingBlockWorkspaceAddr, plancheck.ResourceActionUpdate),
						},
					},
					ConfigStateChecks: []statecheck.StateCheck{
						statecheck.ExpectKnownValue(buildingBlockWorkspaceAddr, tfjsonpath.New("spec").AtMapKey("inputs").AtMapKey("name").AtMapKey("value"), knownvalue.StringExact(`"updated-name"`)),
						statecheck.ExpectKnownValue(buildingBlockWorkspaceAddr, tfjsonpath.New("all_inputs").AtMapKey("size").AtMapKey("value"), knownvalue.StringExact("16")),
						statecheck.ExpectKnownValue(buildingBlockWorkspaceAddr, tfjsonpath.New("status").AtMapKey("status"), knownvalue.StringExact("SUCCEEDED")),
					},
				},
				{
					// Re-draft the BBD → v2 draft (adds the defaulted operator input);
					// version_latest_release still resolves to v1, so the block is a no-op.
					Config:          upgradeConfig,
					ConfigVariables: redraftVars,
					ConfigPlanChecks: resource.ConfigPlanChecks{
						PreApply: []plancheck.PlanCheck{
							plancheck.ExpectResourceAction(bbBbdAddr, plancheck.ResourceActionUpdate),
							plancheck.ExpectResourceAction(buildingBlockWorkspaceAddr, plancheck.ResourceActionNoop),
						},
					},
				},
				{
					// Release v2 and upgrade the cross-workspace block to it. v2 adds a
					// defaulted operator input the config does not supply; the backend applies the default on
					// upgrade, so the block reaches SUCCEEDED (not WAITING) and surfaces it in all_inputs.
					Config:          upgradeConfig,
					ConfigVariables: renamedVars,
					ConfigPlanChecks: resource.ConfigPlanChecks{
						PreApply: []plancheck.PlanCheck{
							plancheck.ExpectResourceAction(bbBbdAddr, plancheck.ResourceActionUpdate),
							plancheck.ExpectResourceAction(buildingBlockWorkspaceAddr, plancheck.ResourceActionUpdate),
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
		vars := With(NewVariablesWithSuffix(acctest.RandString(8)), "terraform_repository_url", terraformTestdataRepoURL(t))
		// Re-draft (creates a v2 draft; version_latest_release still points to v1) and re-release (releases
		// v2). The mock supports the version dance, so the upgrade steps run in both modes.
		redraftVars := With(vars, "bbd_draft", true)
		// Rotate api_key (new secret_value + bumped secret_version) once the BB is on v2 in both modes.
		rotatedVars := With(vars, "secret_value", "rotated-api-key", "secret_version", "2")

		// Step file 16 pins the BBD's latest released version and declares api_key's secret_version, so the
		// rotation step has something to bump.
		config := bbWorkspaceStepConfig(t, 16, "04_sensitive_user_input_bbd", "variables", "secret-variables", "bbd-variables")

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
			statecheck.ExpectKnownValue(buildingBlockSensitiveAddr, tfjsonpath.New("metadata").AtMapKey("uuid"), xknownvalue.NotEmptyString()),
			statecheck.ExpectKnownValue(buildingBlockSensitiveAddr,
				tfjsonpath.New("all_inputs").AtMapKey("api_key").AtMapKey("sensitive").AtMapKey("secret_hash"),
				xknownvalue.NotEmptyString()),
			statecheck.ExpectKnownValue(buildingBlockSensitiveAddr,
				tfjsonpath.New("all_inputs").AtMapKey("script").AtMapKey("sensitive").AtMapKey("secret_hash"),
				xknownvalue.NotEmptyString()),
			statecheck.ExpectKnownValue(buildingBlockSensitiveAddr, tfjsonpath.New("status").AtMapKey("latest_run_uuid"), captureRunUuid),
		}
		if !IsMockClientTest() {
			createChecks = append(createChecks,
				// STATIC sensitive input: the mock does not resolve STATIC inputs from the BBD.
				statecheck.ExpectKnownValue(buildingBlockSensitiveAddr,
					tfjsonpath.New("all_inputs").AtMapKey("static_secret").AtMapKey("sensitive").AtMapKey("secret_hash"),
					xknownvalue.NotEmptyString()),
				// Acceptance-only: with wait_for_completion (default true) the create apply blocks until
				// the run is terminal. Against the real tf-block-runner (cloning the local bare repo and
				// running OpenTofu) this proves the run actually reached SUCCEEDED — the no-op manual
				// runner used to complete it trivially without running terraform.
				statecheck.ExpectKnownValue(buildingBlockSensitiveAddr,
					tfjsonpath.New("status").AtMapKey("status"),
					knownvalue.StringExact("SUCCEEDED")),
			)
		}

		ApplyAndTest(t, resource.TestCase{
			Steps: []resource.TestStep{
				{
					// Step 1: create BBD v1 + BB on v1. Sensitive inputs surface as hashes in all_inputs.
					Config:          config,
					ConfigVariables: vars,
					ConfigPlanChecks: resource.ConfigPlanChecks{
						PreApply: []plancheck.PlanCheck{
							plancheck.ExpectResourceAction(buildingBlockSensitiveAddr, plancheck.ResourceActionCreate),
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
						rs := s.RootModule().Resources[buildingBlockSensitiveAddr]
						if rs == nil {
							return "", fmt.Errorf("resource not found: %s", buildingBlockSensitiveAddr)
						}
						return rs.Primary.Attributes["metadata.uuid"], nil
					},
					ResourceName:    buildingBlockSensitiveAddr,
					ConfigVariables: vars,
				},
				{
					// Step 3: re-draft the BBD → v2 draft. version_latest_release
					// still resolves to v1, so the BB plan is a no-op.
					Config:          config,
					ConfigVariables: redraftVars,
					ConfigPlanChecks: resource.ConfigPlanChecks{
						PreApply: []plancheck.PlanCheck{
							plancheck.ExpectResourceAction(bbSensitiveBbdAddr, plancheck.ResourceActionUpdate),
							plancheck.ExpectResourceAction(buildingBlockSensitiveAddr, plancheck.ResourceActionNoop),
						},
					},
				},
				{
					// Step 4: release BBD v2 + upgrade the BB to v2 in one apply. The
					// sensitive api_key is echoed as its hash sentinel and the secret must survive.
					Config:          config,
					ConfigVariables: vars,
					ConfigPlanChecks: resource.ConfigPlanChecks{
						PreApply: []plancheck.PlanCheck{
							plancheck.ExpectResourceAction(bbSensitiveBbdAddr, plancheck.ResourceActionUpdate),
							plancheck.ExpectResourceAction(buildingBlockSensitiveAddr, plancheck.ResourceActionUpdate),
						},
					},
					ConfigStateChecks: []statecheck.StateCheck{
						statecheck.ExpectKnownValue(buildingBlockSensitiveAddr, tfjsonpath.New("metadata").AtMapKey("uuid"), xknownvalue.NotEmptyString()),
						// Sensitive hash still present after upgrade (secret not corrupted).
						statecheck.ExpectKnownValue(buildingBlockSensitiveAddr,
							tfjsonpath.New("all_inputs").AtMapKey("api_key").AtMapKey("sensitive").AtMapKey("secret_hash"),
							xknownvalue.NotEmptyString()),
						// Re-capture the run uuid so the rotation step below compares against the upgrade run.
						statecheck.ExpectKnownValue(buildingBlockSensitiveAddr, tfjsonpath.New("status").AtMapKey("latest_run_uuid"), captureRunUuid),
					},
				},
				{
					// Step 5: post-upgrade plan must be empty — no spurious rerun and
					// no phantom-input drift.
					Config:          config,
					ConfigVariables: vars,
					PlanOnly:        true,
				},
				{
					// Step 6: rotate api_key (bumped secret_version). Rotation is invisible to the rerun
					// predicate and must be detected via the changed secret_version, so this is an
					// in-place Update and latest_run_uuid must change vs. the previous run.
					Config:          config,
					ConfigVariables: rotatedVars,
					ConfigPlanChecks: resource.ConfigPlanChecks{
						PreApply: []plancheck.PlanCheck{
							plancheck.ExpectResourceAction(buildingBlockSensitiveAddr, plancheck.ResourceActionUpdate),
						},
					},
					ConfigStateChecks: []statecheck.StateCheck{
						statecheck.ExpectKnownValue(buildingBlockSensitiveAddr,
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

		// Step 1 config: admin creates workspace + BBD + other workspace + API key. API key 8 is the
		// consumer's own workspace-scoped key, with no MANAGED_/ADM_ authority.
		step1Config := bbCrossWorkspaceSupportConfig(t, 8, "07_non_updateable")

		// Step 2 config: the "other" provider creates a BB with consumer-only inputs. The BBD marks its
		// `environment` input non-updateable-by-consumer, so step 3 dials that input and must fail.
		vars := NewVariablesWithSuffix(acctest.RandString(8))
		var apiKeyClientId, apiKeyClientSecret lazyVariable
		keyVars := With(vars, "apikey_client_id", &apiKeyClientId, "apikey_client_secret", &apiKeyClientSecret)
		stagingVars := With(keyVars, "bb_environment", "staging")
		step2Config := bbCrossWorkspaceStepConfig(t, 15, 8, "07_non_updateable")

		ApplyAndTest(t, resource.TestCase{
			Steps: []resource.TestStep{
				{
					Config:          step1Config,
					ConfigVariables: vars,
					ConfigStateChecks: []statecheck.StateCheck{
						statecheck.ExpectKnownValue(apiKeyResourceAddr, tfjsonpath.New("status").AtMapKey("client_id"), xknownvalue.NotEmptyString(func(clientId string) error {
							apiKeyClientId = lazyVariable(clientId)
							return nil
						})),
						statecheck.ExpectKnownValue(apiKeyResourceAddr, tfjsonpath.New("status").AtMapKey("client_secret"), xknownvalue.NotEmptyString(func(clientSecret string) error {
							apiKeyClientSecret = lazyVariable(clientSecret)
							return nil
						})),
					},
				},
				{
					Config:          step2Config,
					ConfigVariables: keyVars,
					ConfigPlanChecks: resource.ConfigPlanChecks{
						PreApply: []plancheck.PlanCheck{
							plancheck.ExpectResourceAction(buildingBlockWorkspaceAddr, plancheck.ResourceActionCreate),
						},
					},
					ConfigStateChecks: []statecheck.StateCheck{
						statecheck.ExpectKnownValue(buildingBlockWorkspaceAddr, tfjsonpath.New("spec").AtMapKey("inputs").AtMapKey("environment").AtMapKey("value"), knownvalue.StringExact(`"dev"`)),
					},
				},
				{
					Config:          step2Config,
					ConfigVariables: stagingVars,
					ExpectError:     regexp.MustCompile("you don't have sufficient permissions"),
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
			vars := NewVariablesWithSuffix(acctest.RandString(8))
			// Step files 10 and 11 are step file 1 with a mismatched target ref.
			tenantWithName := bbWorkspaceStepConfig(t, 10, "01_workspace")
			workspaceWithUuid := bbWorkspaceStepConfig(t, 11, "01_workspace")

			ApplyAndTest(t, resource.TestCase{
				Steps: []resource.TestStep{
					{
						// target_ref kind=meshTenant must use uuid, not name.
						Config:          tenantWithName,
						ConfigVariables: vars,
						ExpectError:     regexp.MustCompile(`must not be set when kind`),
					},
					{
						// target_ref kind=meshWorkspace must use name, not uuid.
						Config:          workspaceWithUuid,
						ConfigVariables: vars,
						ExpectError:     regexp.MustCompile(`must not be set when kind`),
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
			vars := NewVariablesWithSuffix(acctest.RandString(8))
			config := bbWorkspaceStepConfig(t, 1, "01_workspace")

			// A STATIC BBD input (region) must not be accepted as a customer/operator input. Step file 12 is
			// step file 1 assigning it.
			invalidInputAssignment := bbWorkspaceStepConfig(t, 12, "01_workspace")

			ApplyAndTest(t, resource.TestCase{
				Steps: []resource.TestStep{
					{
						// Create a valid BB first, then (next step) attempt the invalid STATIC-input
						// assignment as an Update.
						Config:          config,
						ConfigVariables: vars,
						ConfigPlanChecks: resource.ConfigPlanChecks{
							PreApply: []plancheck.PlanCheck{
								plancheck.ExpectResourceAction(buildingBlockWorkspaceAddr, plancheck.ResourceActionCreate),
							},
						},
						ConfigStateChecks: []statecheck.StateCheck{
							statecheck.ExpectKnownValue(buildingBlockWorkspaceAddr, tfjsonpath.New("metadata").AtMapKey("uuid"), xknownvalue.NotEmptyString()),
						},
					},
					{
						// Assigning the STATIC input as a customer input must be rejected.
						Config:          invalidInputAssignment,
						ConfigVariables: vars,
						ExpectError:     regexp.MustCompile("is not defined as a customer or platform-operator input"),
					},
				},
			})
		})
	})

	// 09_purge_on_delete sets purge_on_delete = true so teardown deletes the BB via DELETE /{uuid}/purge.
	// Runs in both modes (create + flag wiring + purge teardown succeeds); the CheckDestroy below carries
	// the mechanism and its acceptance-only 404 assertion.
	t.Run("09_purge_on_delete", func(t *testing.T) {
		vars := NewVariablesWithSuffix(acctest.RandString(8))
		config := bbWorkspaceStepConfig(t, 9, "01_workspace")

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
					Config:          config,
					ConfigVariables: vars,
					ConfigPlanChecks: resource.ConfigPlanChecks{
						PreApply: []plancheck.PlanCheck{
							plancheck.ExpectResourceAction(buildingBlockWorkspaceAddr, plancheck.ResourceActionCreate),
						},
					},
					ConfigStateChecks: []statecheck.StateCheck{
						statecheck.ExpectKnownValue(buildingBlockWorkspaceAddr, tfjsonpath.New("metadata").AtMapKey("uuid"), xknownvalue.NotEmptyString(func(v string) error {
							bbUuid = v
							return nil
						})),
						statecheck.ExpectKnownValue(buildingBlockWorkspaceAddr, tfjsonpath.New("purge_on_delete"), knownvalue.Bool(true)),
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
		vars := NewVariablesWithSuffix(acctest.RandString(8))
		// The definition marks `size` a PLATFORM_OPERATOR_MANUAL_INPUT (name/environment stay user inputs).
		// Step file 7 declares all of them, step file 8 only the operator one.
		fullConfig := bbWorkspaceStepConfig(t, 7, "03_operator_inputs")
		sizeOnlyConfig := bbWorkspaceStepConfig(t, 8, "03_operator_inputs")

		ApplyAndTest(t, resource.TestCase{
			Steps: []resource.TestStep{
				{
					Config:            fullConfig,
					ConfigVariables:   vars,
					ConfigStateChecks: bbv3StateChecks(buildingBlockWorkspaceAddr, "my-workspace-building-block", bbv3SizeEnvInputChecks(buildingBlockWorkspaceAddr)...),
				},
				{
					// Dropping the user inputs is an in-place Update, never a Replace.
					Config:          sizeOnlyConfig,
					ConfigVariables: vars,
					ConfigPlanChecks: resource.ConfigPlanChecks{
						PreApply: []plancheck.PlanCheck{
							plancheck.ExpectResourceAction(buildingBlockWorkspaceAddr, plancheck.ResourceActionUpdate),
						},
					},
					ConfigStateChecks: []statecheck.StateCheck{
						// The operator input stays in spec.inputs (it is declared).
						statecheck.ExpectKnownValue(buildingBlockWorkspaceAddr, tfjsonpath.New("spec").AtMapKey("inputs").AtMapKey("size").AtMapKey("value"), knownvalue.StringExact("16")),
						// The un-declared user inputs are preserved, surfaced read-only in all_inputs.
						statecheck.ExpectKnownValue(buildingBlockWorkspaceAddr, tfjsonpath.New("all_inputs").AtMapKey("name").AtMapKey("value"), knownvalue.StringExact(`"my-name"`)),
						statecheck.ExpectKnownValue(buildingBlockWorkspaceAddr, tfjsonpath.New("all_inputs").AtMapKey("environment").AtMapKey("value"), knownvalue.StringExact(`"dev"`)),
						statecheck.ExpectKnownValue(buildingBlockWorkspaceAddr, tfjsonpath.New("all_inputs").AtMapKey("size").AtMapKey("value"), knownvalue.StringExact("16")),
					},
				},
				{
					// No drift: the omitted user inputs must not reappear as a pending change.
					Config:          sizeOnlyConfig,
					ConfigVariables: vars,
					PlanOnly:        true,
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
		// `broken` ref, whose `tofu apply` fails on a deliberately-false precondition. Each case is fully
		// isolated (its own workspaces, BBD, key) so the two opposite outcomes cannot interfere.
		//
		// A run that does not succeed never errors on its own (#277), so what fails these applies is the
		// postcondition the test-support file carries — the recipe the resource documentation recommends,
		// exercised end to end. Each step's plan check is the other half of the assertion: a postcondition
		// is evaluated after the building block is written to state and after every taint call site, so the
		// block must still plan as an in-place update after an apply it failed. A replace there would mean
		// the next apply destroys everything the runs created, which is the whole point of #277.
		//
		// What run_transparency gates for a workspace-scoped key is whether the repair may be triggered at
		// all: with it ON the re-run starts and fails on the broken ref again, with it OFF meshStack
		// refuses the trigger-run and that refusal is the error instead.
		//
		// Two things this case deliberately does not assert. That an unsuccessful run is a warning rather
		// than an error is TestAwaitRun's job. So is the failing step's log reaching the diagnostics: it is
		// a warning now, Terraform prints warnings to stdout, and the framework matches ExpectError against
		// stderr.
		runCase := func(t *testing.T, runTransparency bool, expectRepairErr *regexp.Regexp) {
			t.Helper()
			// Workspace W owns the BBD; workspace O consumes it across the workspace boundary with its own
			// unprivileged, workspace-scoped key (API key 8).
			vars := With(NewVariablesWithSuffix(acctest.RandString(8)), "terraform_repository_url", terraformTestdataRepoURL(t), "run_transparency", runTransparency)

			step1Config := bbCrossWorkspaceSupportConfig(t, 8, "11_broken_run_bbd", "variables", "transparency-variables")
			// The consumer creates the BB via the meshstack-other provider (its workspace-scoped key).
			consumerConfig := bbCrossWorkspaceStepConfig(t, 17, 8, "11_broken_run_bbd", "variables", "transparency-variables")

			var apiKeyClientId, apiKeyClientSecret lazyVariable
			consumerVariables := With(vars, "apikey_client_id", &apiKeyClientId, "apikey_client_secret", &apiKeyClientSecret)
			// The support file's postcondition rejects the FAILED status the broken ref produces. With run
			// transparency ON the repair runs and fails on the broken ref again, so the postcondition fails
			// that apply too; with it OFF meshStack refuses the trigger-run and that refusal is the error.
			postconditionFailed := regexp.MustCompile("not SUCCEEDED")
			repairApplyErr := postconditionFailed
			if expectRepairErr != nil {
				repairApplyErr = expectRepairErr
			}

			steps := []resource.TestStep{
				{
					// Admin mints the infra + the workspace-scoped key; capture its credentials.
					Config:          step1Config,
					ConfigVariables: vars,
					ConfigStateChecks: []statecheck.StateCheck{
						statecheck.ExpectKnownValue(apiKeyResourceAddr, tfjsonpath.New("status").AtMapKey("client_id"), xknownvalue.NotEmptyString(func(clientId string) error {
							apiKeyClientId = lazyVariable(clientId)
							return nil
						})),
						statecheck.ExpectKnownValue(apiKeyResourceAddr, tfjsonpath.New("status").AtMapKey("client_secret"), xknownvalue.NotEmptyString(func(clientSecret string) error {
							apiKeyClientSecret = lazyVariable(clientSecret)
							return nil
						})),
					},
				},
				{
					// The consumer creates the BB; its run fails on the broken ref. wait_for_completion is
					// set in the support file, so the create waits for that run, keeps the building block
					// with a warning, and the postcondition is what turns that into a failed apply.
					Config:          consumerConfig,
					ConfigVariables: consumerVariables,
					ConfigPlanChecks: resource.ConfigPlanChecks{
						PreApply: []plancheck.PlanCheck{
							plancheck.ExpectResourceAction(buildingBlockSensitiveAddr, plancheck.ResourceActionCreate),
						},
					},
					ExpectError: postconditionFailed,
				},
				{
					// Re-applying the unchanged config is the repair. The plan check is the assertion this
					// issue is about: the create failed its apply, and the building block is still an
					// in-place update rather than a replace, so nothing it created is about to be destroyed.
					Config:          consumerConfig,
					ConfigVariables: consumerVariables,
					ConfigPlanChecks: resource.ConfigPlanChecks{
						PreApply: []plancheck.PlanCheck{
							plancheck.ExpectResourceAction(buildingBlockSensitiveAddr, plancheck.ResourceActionUpdate),
						},
					},
					ExpectError: repairApplyErr,
				},
			}

			if expectRepairErr == nil {
				steps = append(steps, resource.TestStep{
					// The repair's apply failed on the postcondition as well, and the block must still plan
					// as an in-place update. A PlanOnly step skips the pre-apply plan, so the check has to
					// be a post-refresh one, and the pending repair is why the plan is not empty.
					Config:          consumerConfig,
					ConfigVariables: consumerVariables,
					ConfigPlanChecks: resource.ConfigPlanChecks{
						PostApplyPostRefresh: []plancheck.PlanCheck{
							plancheck.ExpectResourceAction(buildingBlockSensitiveAddr, plancheck.ResourceActionUpdate),
						},
					},
					PlanOnly:           true,
					ExpectNonEmptyPlan: true,
				})
			}

			ApplyAndTest(t, resource.TestCase{Steps: steps})
		}

		// Run transparency ON: the workspace-scoped key may trigger the repair, so the re-run happens,
		// fails on the broken ref again, and the block keeps planning as an in-place repair.
		t.Run("transparency_on_repairs_in_place", func(t *testing.T) {
			runCase(t, true, nil)
		})
		// Run transparency OFF: meshStack refuses a workspace-scoped key's trigger-run on an opaque
		// definition, so the repair cannot start at all and the apply says so.
		t.Run("transparency_off_refuses_the_repair", func(t *testing.T) {
			runCase(t, false, regexp.MustCompile("meshStack refused to run it again"))
		})
	})

	// 12_moved_from_v2_with_secret guards the v2→v3 migration of a block with a sensitive USER_INPUT: the
	// move must plan in-place AND preserve the secret, even though secret_value is write-only and cannot
	// ride through state (moveFromV2 + secret.ValueToConverter handle the seed/refresh/echo). The v3
	// config re-declares the input with a DISTINCT placeholder and no secret_version; corruption is
	// detectable because sending the placeholder plaintext would change the all_inputs hash. Runs in both
	// modes (the mock's backendSecretBehavior mirrors the backend).
	t.Run("12_moved_from_v2_with_secret", func(t *testing.T) {
		vars := With(NewVariablesWithSuffix(acctest.RandString(8)), "terraform_repository_url", terraformTestdataRepoURL(t))
		// Rotation: bump secret_version (null->"2") + new value, so the secret is re-applied and its hash
		// changes.
		rotatedVars := With(vars, "moved_secret_value", "rotated-real-api-key", "moved_secret_version", "2")

		// Shared sensitive BBD (api_key STRING + script CODE USER_INPUTs, static_secret STATIC), pointed at
		// the committed bare repo so acceptance runs execute quickly (unused in mock mode).
		bbdConfig := examples.JoinTestStepConfigs(
			examples.Resource.TestSupportConfigs(t, "building_block", "04_sensitive_user_input_bbd", "variables", "bbd-variables"),
			examples.Resource.TestStepConfig(t, "workspace", 1, "variables", "prerequisites"),
		)
		// v2 block supplying the real secrets via value_string_sensitive/value_code_sensitive.
		v2Config := examples.JoinTestStepConfigs(
			examples.Resource.TestStepConfig(t, "building_block_v2", 5),
			bbdConfig,
		)
		// v3 block: same definition + target; re-declares the secrets with DISTINCT placeholders, no version.
		v3Config := examples.JoinTestStepConfigs(
			examples.Resource.TestStepConfig(t, "building_block", 19, "moved-secret-variables"),
			bbdConfig,
		)
		movedConfig := examples.Resource.TestSupportConfigs(t, "building_block", "moved_from_v2_secret")

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
					Config:          v2Config,
					ConfigVariables: vars,
					ConfigPlanChecks: resource.ConfigPlanChecks{
						PreApply: []plancheck.PlanCheck{plancheck.ExpectResourceAction(bbv2MovedSecretAddr, plancheck.ResourceActionCreate)},
					},
					ConfigStateChecks: []statecheck.StateCheck{
						statecheck.ExpectKnownValue(bbv2MovedSecretAddr, tfjsonpath.New("metadata").AtMapKey("uuid"), xknownvalue.NotEmptyString()),
						statecheck.ExpectKnownValue(bbv2MovedSecretAddr, tfjsonpath.New("spec").AtMapKey("combined_inputs").AtMapKey("api_key").AtMapKey("value_string"), captureApiKeyHash),
						statecheck.ExpectKnownValue(bbv2MovedSecretAddr, tfjsonpath.New("spec").AtMapKey("combined_inputs").AtMapKey("script").AtMapKey("value_code"), captureScriptHash),
					},
				},
				{
					// The move must plan as an in-place Update (never Replace), and the secrets must be
					// preserved: the v3 all_inputs hashes must equal the v2 hashes despite the placeholders.
					Config:          examples.JoinTestStepConfigs(v3Config, movedConfig),
					ConfigVariables: vars,
					ConfigPlanChecks: resource.ConfigPlanChecks{
						PreApply: []plancheck.PlanCheck{plancheck.ExpectResourceAction(buildingBlockMovedSecretAddr, plancheck.ResourceActionUpdate)},
					},
					ConfigStateChecks: []statecheck.StateCheck{
						statecheck.ExpectKnownValue(buildingBlockMovedSecretAddr,
							tfjsonpath.New("all_inputs").AtMapKey("api_key").AtMapKey("sensitive").AtMapKey("secret_hash"),
							xknownvalue.NotEmptyString(func(v string) error {
								if v != v2ApiKeyHash {
									return fmt.Errorf("api_key secret was not preserved across the move: hash %q != v2 hash %q (the re-supplied placeholder overwrote the secret)", v, v2ApiKeyHash)
								}
								return nil
							})),
						statecheck.ExpectKnownValue(buildingBlockMovedSecretAddr,
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
					Config:          examples.JoinTestStepConfigs(v3Config, movedConfig),
					ConfigVariables: rotatedVars,
					ConfigStateChecks: []statecheck.StateCheck{
						statecheck.ExpectKnownValue(buildingBlockMovedSecretAddr,
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

	// 13_parent_child is the only scenario with a real parent. Every other parent step uses a
	// synthetic uuid and therefore never reaches a live backend.
	t.Run("13_parent_child", func(t *testing.T) {
		vars := NewVariablesWithSuffix(acctest.RandString(8))
		// Both blocks come from the same definition, so each one is its own step file.
		config := examples.JoinTestStepConfigs(
			examples.Resource.TestStepConfig(t, "building_block", 3),
			bbWorkspaceStepConfig(t, 2, "01_workspace"),
		)

		ApplyAndTest(t, resource.TestCase{
			Steps: []resource.TestStep{
				{
					Config:          config,
					ConfigVariables: vars,
					ConfigPlanChecks: resource.ConfigPlanChecks{
						PreApply: []plancheck.PlanCheck{
							plancheck.ExpectResourceAction(buildingBlockParentAddr, plancheck.ResourceActionCreate),
							plancheck.ExpectResourceAction(buildingBlockChildAddr, plancheck.ResourceActionCreate),
						},
					},
					ConfigStateChecks: append(
						bbv3StateChecks(buildingBlockChildAddr, "my-child-building-block"),
						statecheck.ExpectKnownValue(buildingBlockChildAddr, tfjsonpath.New("spec").AtMapKey("parent_building_block_refs"), knownvalue.SetSizeExact(1)),
						statecheck.CompareValuePairs(
							buildingBlockParentAddr, tfjsonpath.New("ref"),
							buildingBlockChildAddr, tfjsonpath.New("spec").AtMapKey("parent_building_block_refs").AtSliceIndex(0),
							compare.ValuesSame(),
						),
					),
				},
				{
					// The parent ref read back must hash to the same set element as the one the
					// configuration declares, or this plan is not empty.
					Config:          config,
					ConfigVariables: vars,
					PlanOnly:        true,
				},
			},
		})
	})
}

// bbv3StateChecks returns the baseline state checks shared by every BB v3 create and move step.
func bbv3StateChecks(buildingBlockAddr, displayName string, extra ...statecheck.StateCheck) []statecheck.StateCheck {
	checks := []statecheck.StateCheck{
		statecheck.ExpectKnownValue(buildingBlockAddr, tfjsonpath.New("metadata").AtMapKey("uuid"), xknownvalue.NotEmptyString()),
		xknownvalue.Ref(buildingBlockAddr, client.MeshObjectKind.BuildingBlock, nil),
		statecheck.ExpectKnownValue(buildingBlockAddr, tfjsonpath.New("spec").AtMapKey("display_name"), knownvalue.StringExact(displayName)),
		statecheck.ExpectKnownValue(buildingBlockAddr, tfjsonpath.New("spec").AtMapKey("inputs").AtMapKey("name").AtMapKey("value"), knownvalue.StringExact(`"my-name"`)),
		statecheck.ExpectKnownValue(buildingBlockAddr, tfjsonpath.New("status").AtMapKey("status"), knownvalue.StringExact("SUCCEEDED")),
		statecheck.ExpectKnownValue(buildingBlockAddr, tfjsonpath.New("status").AtMapKey("latest_run_uuid"), xknownvalue.NotEmptyString()),
	}
	return append(checks, extra...)
}

// bbv3SizeEnvInputChecks are the size/environment input-value checks shared by the workspace and
// tenant examples (resource_01_workspace.tf / resource_02_tenant.tf) and the moved-from-v2 scenario.
func bbv3SizeEnvInputChecks(buildingBlockAddr string) []statecheck.StateCheck {
	return []statecheck.StateCheck{
		statecheck.ExpectKnownValue(buildingBlockAddr, tfjsonpath.New("spec").AtMapKey("inputs").AtMapKey("size").AtMapKey("value"), knownvalue.StringExact("16")),
		statecheck.ExpectKnownValue(buildingBlockAddr, tfjsonpath.New("spec").AtMapKey("inputs").AtMapKey("environment").AtMapKey("value"), knownvalue.StringExact(`"dev"`)),
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

	spec := func(uuid string, contentHash *string, inputs map[string]*client.MeshBuildingBlockInput, parents ...client.UuidRef) client.MeshBuildingBlockV2Spec {
		return client.MeshBuildingBlockV2Spec{
			BuildingBlockDefinitionVersionRef: client.MeshBuildingBlockV2DefinitionVersionRef{
				UuidRef:     client.UuidRef{Uuid: uuid},
				ContentHash: contentHash,
			},
			Inputs:                  inputs,
			ParentBuildingBlockRefs: parents,
		}
	}
	input := func(sensitive bool) *client.MeshBuildingBlockInput {
		return &client.MeshBuildingBlockInput{IsSensitive: sensitive}
	}
	parent := client.UuidRef{Kind: client.MeshObjectKind.BuildingBlock, Uuid: "parent-1"}

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
