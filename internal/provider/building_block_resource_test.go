package provider

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/compare"
	tfconfig "github.com/hashicorp/terraform-plugin-testing/config"
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

	"github.com/meshcloud/terraform-provider-meshstack/internal/provider/acctest/xknownvalue"
)

// The runner shares the test's filesystem, so it clones the repo by path.
func terraformTestdataRepoURL(t *testing.T) string {
	t.Helper()
	repo := t.TempDir()
	git := func(args ...string) {
		t.Helper()
		cmd := exec.Command("git", append([]string{"-C", repo}, args...)...)
		cmd.Env = append(os.Environ(),
			"GIT_CONFIG_GLOBAL="+os.DevNull, "GIT_CONFIG_NOSYSTEM=1",
			"GIT_AUTHOR_NAME=acctest", "GIT_AUTHOR_EMAIL=acctest@example.com",
			"GIT_COMMITTER_NAME=acctest", "GIT_COMMITTER_EMAIL=acctest@example.com",
		)
		out, err := cmd.CombinedOutput()
		require.NoError(t, err, "git %v: %s", args, out)
	}
	commit := func(dir string) {
		t.Helper()
		require.NoError(t, os.CopyFS(repo, os.DirFS(filepath.Join("testdata/tf-building-block", dir))))
		git("add", ".")
		git("commit", "-m", dir)
	}
	git("init", "-b", "main")
	commit("main")
	git("switch", "-c", "broken")
	commit("broken")
	git("switch", "main")
	return "file://" + repo
}

// The subtests below are scenario flows rather than one-assertion-per-case tests: each walks a
// building block through a sequence of steps. Every flow runs in both modes; where an assertion
// only holds against the real backend the ConfigStateChecks are gated on IsMockClientTest(), and
// where a whole flow needs the real backend it lives in its own subtest that t.Skip()s in mock
// (see 11). See the lock-step policy on IsMockClientTest.

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

func TestAccBuildingBlock(t *testing.T) {
	t.Parallel()

	// 01_workspace_lifecycle: full workspace-BB life — create, import, in-place updates (display_name,
	// input value, content_hash), and a parent change that forces a replace. Step 8 (parent→Replace) is
	// mock-only: it asserts a provider-side plan decision (RequiresReplaceIf), and the real backend
	// rejects the synthetic parent-BB UUID before the plan can be observed.
	t.Run("01_workspace_lifecycle", func(t *testing.T) {
		suffix := acctest.RandString(8)
		changedContentHash := BuildingBlockDefinitionVersionContentHash{hashVersion: currentHashVersion - 1, hashValue: "aaa"}.toBase64()
		rehashedContentHash := BuildingBlockDefinitionVersionContentHash{hashVersion: currentHashVersion, hashValue: "aaa"}.toBase64()
		vars := hclStepVars{"suffix": suffix, "changed_content_hash": changedContentHash, "rehashed_content_hash": rehashedContentHash}
		buildingBlockAddr := "meshstack_building_block.example_workspace"

		ApplyAndTest(t, resource.TestCase{
			Steps: []resource.TestStep{
				{
					Config: hclStep(t, "resources/building_block/01_workspace_lifecycle", 1, vars),
					ConfigPlanChecks: resource.ConfigPlanChecks{
						PreApply: []plancheck.PlanCheck{
							plancheck.ExpectResourceAction(buildingBlockAddr, plancheck.ResourceActionCreate),
						},
					},
					ConfigStateChecks: bbv3StateChecks(buildingBlockAddr, "my-workspace-building-block",
						append(bbv3SizeEnvInputChecks(buildingBlockAddr), bbv3PaymentMethodInputChecks(buildingBlockAddr, suffix)...)...),
				},
				{
					// content_hash never comes back from the API; purge_on_delete and timeouts exist only in config.
					ImportState:                          true,
					ImportStateVerify:                    true,
					ImportStateVerifyIdentifierAttribute: "metadata.uuid",
					ImportStateVerifyIgnore:              []string{"spec.building_block_definition_version_ref.content_hash", "purge_on_delete", "timeouts.create", "timeouts.update", "timeouts.delete"},
					ImportStateIdFunc: func(s *terraform.State) (string, error) {
						rs := s.RootModule().Resources[buildingBlockAddr]
						if rs == nil {
							return "", fmt.Errorf("resource not found: %s", buildingBlockAddr)
						}
						return rs.Primary.Attributes["metadata.uuid"], nil
					},
					ResourceName: buildingBlockAddr,
				},
				{
					// The refreshed plan must be empty: unconfigured optional USER_INPUTs that the backend
					// echoes as null rows must not surface as drift. PlanOnly without ExpectNonEmptyPlan
					// asserts an empty plan. Runs in both modes — the mock materializes the same null rows,
					// so the check holds there too and we keep mock/acceptance behaviour in lock-step.
					Config:   hclStep(t, "resources/building_block/01_workspace_lifecycle", 3, vars),
					PlanOnly: true,
				},
				{
					// Rename only display_name → in-place Update, never Replace.
					Config: hclStep(t, "resources/building_block/01_workspace_lifecycle", 4, vars),
					ConfigPlanChecks: resource.ConfigPlanChecks{
						PreApply: []plancheck.PlanCheck{
							plancheck.ExpectResourceAction(buildingBlockAddr, plancheck.ResourceActionUpdate),
						},
					},
					ConfigStateChecks: []statecheck.StateCheck{
						statecheck.ExpectKnownValue(buildingBlockAddr, tfjsonpath.New("spec").AtMapKey("display_name"), knownvalue.StringExact("my-workspace-building-block-renamed")),
					},
				},
				{
					// Change an input value → in-place Update.
					// jsonencode escapes `&` as `\u0026`, while meshStack returns the value unescaped.
					Config: hclStep(t, "resources/building_block/01_workspace_lifecycle", 5, vars),
					ConfigPlanChecks: resource.ConfigPlanChecks{
						PreApply: []plancheck.PlanCheck{
							plancheck.ExpectResourceAction(buildingBlockAddr, plancheck.ResourceActionUpdate),
						},
					},
					ConfigStateChecks: []statecheck.StateCheck{
						statecheck.ExpectKnownValue(buildingBlockAddr, tfjsonpath.New("spec").AtMapKey("inputs").AtMapKey("environment").AtMapKey("value"), knownvalue.StringExact(`"staging"`)),
						statecheck.ExpectKnownValue(buildingBlockAddr, tfjsonpath.New("spec").AtMapKey("inputs").AtMapKey("name").AtMapKey("value"), knownvalue.StringExact(`"R\u0026D"`)),
					},
				},
				{
					// Set initial content_hash to track BBD version "v1".
					Config: hclStep(t, "resources/building_block/01_workspace_lifecycle", 6, vars),
					ConfigPlanChecks: resource.ConfigPlanChecks{
						PreApply: []plancheck.PlanCheck{
							plancheck.ExpectResourceAction(buildingBlockAddr, plancheck.ResourceActionUpdate),
						},
					},
					ConfigStateChecks: []statecheck.StateCheck{
						statecheck.ExpectKnownValue(buildingBlockAddr, tfjsonpath.New("spec").AtMapKey("building_block_definition_version_ref").AtMapKey("content_hash"), knownvalue.StringExact("v1")),
					},
				},
				{
					// Changing content_hash simulates a BBD content update and triggers a rerun.
					Config: hclStep(t, "resources/building_block/01_workspace_lifecycle", 7, vars),
					ConfigPlanChecks: resource.ConfigPlanChecks{
						PreApply: []plancheck.PlanCheck{
							plancheck.ExpectResourceAction(buildingBlockAddr, plancheck.ResourceActionUpdate),
						},
					},
					ConfigStateChecks: []statecheck.StateCheck{
						statecheck.ExpectKnownValue(buildingBlockAddr, tfjsonpath.New("spec").AtMapKey("building_block_definition_version_ref").AtMapKey("content_hash"), knownvalue.StringExact(changedContentHash)),
					},
				},
				{
					// A hash version change alone plans no run on a released version.
					Config: hclStep(t, "resources/building_block/01_workspace_lifecycle", 8, vars),
					ConfigPlanChecks: resource.ConfigPlanChecks{
						PreApply: []plancheck.PlanCheck{
							plancheck.ExpectResourceAction(buildingBlockAddr, plancheck.ResourceActionUpdate),
							plancheck.ExpectKnownValue(buildingBlockAddr, tfjsonpath.New("status").AtMapKey("action"),
								knownvalue.StringExact(client.BuildingBlockActionNoAction.String())),
						},
					},
					ConfigStateChecks: []statecheck.StateCheck{
						statecheck.ExpectKnownValue(buildingBlockAddr, tfjsonpath.New("status").AtMapKey("action"),
							knownvalue.StringExact(client.BuildingBlockActionNoAction.String())),
					},
				},
				{
					Config: hclStep(t, "resources/building_block/01_workspace_lifecycle", 9, vars),
					ConfigPlanChecks: resource.ConfigPlanChecks{
						PreApply: []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()},
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
					Config: hclStep(t, "resources/building_block/01_workspace_lifecycle", 10, vars),
					ConfigPlanChecks: resource.ConfigPlanChecks{
						PreApply: []plancheck.PlanCheck{
							plancheck.ExpectResourceAction(buildingBlockAddr, plancheck.ResourceActionDestroyBeforeCreate),
						},
					},
				},
			},
		})
	})

	// The mock hashes sensitive plaintext like the backend, so the hash check runs in both modes.
	t.Run("02_tenant", func(t *testing.T) {
		vars := hclStepVars{"suffix": acctest.RandString(8), "repository_url": terraformTestdataRepoURL(t)}
		buildingBlockAddr := "meshstack_building_block.example_tenant"

		// api_key (sensitive STRING USER_INPUT) surfaces as a non-empty hash in all_inputs in both modes.
		sensitiveInputChecks := append(bbv3SizeEnvInputChecks(buildingBlockAddr),
			statecheck.ExpectKnownValue(buildingBlockAddr,
				tfjsonpath.New("all_inputs").AtMapKey("api_key").AtMapKey("sensitive").AtMapKey("secret_hash"),
				xknownvalue.NotEmptyString()),
		)
		if !IsMockClientTest() {
			// Only the runner decrypts api_key, which the module echoes back as a JSON-encoded output.
			sensitiveInputChecks = append(sensitiveInputChecks,
				statecheck.ExpectKnownValue(buildingBlockAddr,
					tfjsonpath.New("status").AtMapKey("outputs").AtMapKey("api_key_echo").AtMapKey("value"),
					knownvalue.StringExact(`"super-secret-api-key"`)),
			)
		}

		ApplyAndTest(t, resource.TestCase{
			Steps: []resource.TestStep{
				{
					Config: hclStep(t, "resources/building_block/02_tenant", 1, vars),
					ConfigPlanChecks: resource.ConfigPlanChecks{
						PreApply: []plancheck.PlanCheck{
							plancheck.ExpectResourceAction(buildingBlockAddr, plancheck.ResourceActionCreate),
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
						"purge_on_delete",
						"timeouts.create",
						"timeouts.update",
						"timeouts.delete",
					},
					ImportStateIdFunc: func(s *terraform.State) (string, error) {
						rs := s.RootModule().Resources[buildingBlockAddr]
						if rs == nil {
							return "", fmt.Errorf("resource not found: %s", buildingBlockAddr)
						}
						return rs.Primary.Attributes["metadata.uuid"], nil
					},
					ResourceName: buildingBlockAddr,
				},
			},
		})
	})

	// 05_operator_inputs walks a platform-operator input (`size`, PLATFORM_OPERATOR_MANUAL_INPUT) through
	// its lifecycle on a cross-workspace BB: create without it (parks with the action PROVIDE_INPUT), resume
	// via a PUT that supplies it (the provider must wait past the PROVIDE_INPUT the PUT answers with), update
	// a consumer input, then upgrade to a v2 BBD that adds a defaulted operator input. All steps run in
	// both modes; only two assertions are backend-gated (the mock never waits for input, and does not
	// materialize the v2 default `tier`). A MANAGED_BUILDINGBLOCK_SAVE key is minted to prove the provider
	// accepts that authority; its cross-workspace authorization is covered on the backend by MeshBuildingBlockManagedSaveScenarios.
	t.Run("05_operator_inputs", func(t *testing.T) {
		// Workspace A owns the BBD (which declares `size` as PLATFORM_OPERATOR_MANUAL_INPUT) and
		// the API key used to set the operator input.
		// Workspace B is the consumer: the building block lives here, across the workspace boundary
		// from the definition owner.
		// Key owned by workspace A (the definition owner). MANAGED_BUILDINGBLOCK_SAVE is the capability
		// under test: it lets the key create and update building blocks consuming its definition in
		// other workspaces (the key deliberately has no ADM_BUILDINGBLOCK_SAVE, so every create/update
		// below is authorized purely by the MANAGED scope). ADM_BUILDINGBLOCK_DELETE is present only so
		// the meshstack-other provider can tear the cross-workspace block down at end of test: there is
		// no MANAGED_BUILDINGBLOCK_DELETE authority, so a platform operator can save but not delete a
		// block it doesn't own — delete is not the capability being exercised here.
		// Step 1 sets up the infrastructure and mints the key (default/admin provider).
		// Steps 2+ keep the meshstack-other provider alias configured (the MANAGED key minted above); the
		// block itself is admin-managed, so every step passes the key credentials as config variables.
		// The example's postcondition would fail the apply of a block that waits for an operator input.

		// v2 of the definition adds a defaulted platform-operator input `tier`. Re-draft (new v2 draft;
		// version_latest_release still v1) then re-release (v2 released) mirror the version dance in 06, so
		// the upgrade steps below prove the backend applies an operator-input default on upgrade.
		vars := hclStepVars{"suffix": acctest.RandString(8)}
		buildingBlockDefinitionAddr := "meshstack_building_block_definition.example"
		apiKeyAddr := "meshstack_api_key.reader"
		buildingBlockAddr := "meshstack_building_block.example_workspace"

		// The mock never waits for input, so its create finishes the run at once.
		createStatusChecks := []statecheck.StateCheck{
			statecheck.ExpectKnownValue(buildingBlockAddr, tfjsonpath.New("status").AtMapKey("health_status"), knownvalue.StringExact("NONE")),
			statecheck.ExpectKnownValue(buildingBlockAddr, tfjsonpath.New("status").AtMapKey("action"), knownvalue.StringExact("PROVIDE_INPUT")),
			statecheck.ExpectKnownValue(buildingBlockAddr, tfjsonpath.New("status").AtMapKey("succeeded"), knownvalue.Bool(false)),
		}
		if IsMockClientTest() {
			createStatusChecks = []statecheck.StateCheck{
				statecheck.ExpectKnownValue(buildingBlockAddr, tfjsonpath.New("status").AtMapKey("action"), xknownvalue.NotEmptyString()),
			}
		}

		// Post-upgrade checks: the block reaches SUCCEEDED in both modes. The `tier` assertion is
		// acceptance-only — applying a defaulted operator input on upgrade is backend behaviour the
		// mock does not reproduce (it neither runs the block nor materializes the added default).
		upgradedChecks := []statecheck.StateCheck{
			statecheck.ExpectKnownValue(buildingBlockAddr, tfjsonpath.New("status").AtMapKey("succeeded"), knownvalue.Bool(true)),
		}
		if !IsMockClientTest() {
			upgradedChecks = append(upgradedChecks,
				statecheck.ExpectKnownValue(buildingBlockAddr, tfjsonpath.New("all_inputs").AtMapKey("tier").AtMapKey("value"), knownvalue.StringExact(`"bronze"`)),
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
					Config: hclStep(t, "resources/building_block/05_operator_inputs", 1, vars),
					ConfigStateChecks: []statecheck.StateCheck{
						statecheck.ExpectKnownValue(apiKeyAddr, tfjsonpath.New("status").AtMapKey("client_id"), xknownvalue.NotEmptyString(func(clientId string) error {
							apiKeyClientId = lazyVariable(clientId)
							return nil
						})),
						statecheck.ExpectKnownValue(apiKeyAddr, tfjsonpath.New("status").AtMapKey("client_secret"), xknownvalue.NotEmptyString(func(clientSecret string) error {
							apiKeyClientSecret = lazyVariable(clientSecret)
							return nil
						})),
					},
				},
				{
					// Create without the operator input: on a real backend the block parks with the action
					// PROVIDE_INPUT (the provider surfaces a warning, not an error).
					Config:          hclStep(t, "resources/building_block/05_operator_inputs", 2, vars),
					ConfigVariables: creds,
					ConfigPlanChecks: resource.ConfigPlanChecks{
						PreApply: []plancheck.PlanCheck{
							plancheck.ExpectResourceAction(buildingBlockAddr, plancheck.ResourceActionCreate),
						},
					},
					ConfigStateChecks: append([]statecheck.StateCheck{
						statecheck.ExpectKnownValue(buildingBlockAddr, tfjsonpath.New("spec").AtMapKey("inputs").AtMapKey("name").AtMapKey("value"), knownvalue.StringExact(`"my-name"`)),
					}, createStatusChecks...),
				},
				{
					// Supplying `size` via PUT resumes provisioning; the provider must poll THROUGH the
					// transient WAITING to the resulting SUCCEEDED run instead of returning on the stale WAITING.
					Config:          hclStep(t, "resources/building_block/05_operator_inputs", 3, vars),
					ConfigVariables: creds,
					ConfigPlanChecks: resource.ConfigPlanChecks{
						PreApply: []plancheck.PlanCheck{
							plancheck.ExpectResourceAction(buildingBlockAddr, plancheck.ResourceActionUpdate),
						},
					},
					ConfigStateChecks: []statecheck.StateCheck{
						statecheck.ExpectKnownValue(buildingBlockAddr, tfjsonpath.New("all_inputs").AtMapKey("size").AtMapKey("value"), knownvalue.StringExact("16")),
						statecheck.ExpectKnownValue(buildingBlockAddr, tfjsonpath.New("status").AtMapKey("health_status"), knownvalue.StringExact("SUCCEEDED")),
						statecheck.ExpectKnownValue(buildingBlockAddr, tfjsonpath.New("status").AtMapKey("action"), knownvalue.StringExact("NO_ACTION")),
						statecheck.ExpectKnownValue(buildingBlockAddr, tfjsonpath.New("status").AtMapKey("succeeded"), knownvalue.Bool(true)),
						statecheck.ExpectKnownValue(buildingBlockAddr, tfjsonpath.New("status").AtMapKey("latest_run_uuid"), xknownvalue.NotEmptyString()),
					},
				},
				{
					// Changing a consumer input is an in-place Update; the operator input stays put.
					Config:          hclStep(t, "resources/building_block/05_operator_inputs", 4, vars),
					ConfigVariables: creds,
					ConfigPlanChecks: resource.ConfigPlanChecks{
						PreApply: []plancheck.PlanCheck{
							plancheck.ExpectResourceAction(buildingBlockAddr, plancheck.ResourceActionUpdate),
						},
					},
					ConfigStateChecks: []statecheck.StateCheck{
						statecheck.ExpectKnownValue(buildingBlockAddr, tfjsonpath.New("spec").AtMapKey("inputs").AtMapKey("name").AtMapKey("value"), knownvalue.StringExact(`"updated-name"`)),
						statecheck.ExpectKnownValue(buildingBlockAddr, tfjsonpath.New("all_inputs").AtMapKey("size").AtMapKey("value"), knownvalue.StringExact("16")),
						statecheck.ExpectKnownValue(buildingBlockAddr, tfjsonpath.New("status").AtMapKey("succeeded"), knownvalue.Bool(true)),
					},
				},
				{
					// Re-draft the BBD → v2 draft (adds the defaulted operator input);
					// version_latest_release still resolves to v1, so the block is a no-op.
					Config:          hclStep(t, "resources/building_block/05_operator_inputs", 5, vars),
					ConfigVariables: creds,
					ConfigPlanChecks: resource.ConfigPlanChecks{
						PreApply: []plancheck.PlanCheck{
							plancheck.ExpectResourceAction(buildingBlockDefinitionAddr, plancheck.ResourceActionUpdate),
							plancheck.ExpectResourceAction(buildingBlockAddr, plancheck.ResourceActionNoop),
						},
					},
				},
				{
					// Release v2 and upgrade the cross-workspace block to it. v2 adds a
					// defaulted operator input the config does not supply; the backend applies the default on
					// upgrade, so the block reaches SUCCEEDED (not WAITING) and surfaces it in all_inputs.
					Config:          hclStep(t, "resources/building_block/05_operator_inputs", 6, vars),
					ConfigVariables: creds,
					ConfigPlanChecks: resource.ConfigPlanChecks{
						PreApply: []plancheck.PlanCheck{
							plancheck.ExpectResourceAction(buildingBlockDefinitionAddr, plancheck.ResourceActionUpdate),
							plancheck.ExpectResourceAction(buildingBlockAddr, plancheck.ResourceActionUpdate),
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
		// Re-draft (creates a v2 draft; version_latest_release still points to v1) and re-release
		// (releases v2). The mock supports the version dance, so the upgrade steps run in both modes.
		// Build a BB pinned to the BBD's latest released version, supplying api_key + script secrets
		// and pinning api_key.secret_version so the rotation step can bump it.
		vars := hclStepVars{"suffix": acctest.RandString(8), "repository_url": terraformTestdataRepoURL(t)}
		bbdAddr := "meshstack_building_block_definition.sensitive_user_input"
		bbAddr := "meshstack_building_block.sensitive_user_input"

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
			statecheck.ExpectKnownValue(bbAddr, tfjsonpath.New("metadata").AtMapKey("uuid"), xknownvalue.NotEmptyString()),
			statecheck.ExpectKnownValue(bbAddr,
				tfjsonpath.New("all_inputs").AtMapKey("api_key").AtMapKey("sensitive").AtMapKey("secret_hash"),
				xknownvalue.NotEmptyString()),
			statecheck.ExpectKnownValue(bbAddr,
				tfjsonpath.New("all_inputs").AtMapKey("script").AtMapKey("sensitive").AtMapKey("secret_hash"),
				xknownvalue.NotEmptyString()),
			statecheck.ExpectKnownValue(bbAddr, tfjsonpath.New("status").AtMapKey("latest_run_uuid"), captureRunUuid),
		}
		if !IsMockClientTest() {
			createChecks = append(createChecks,
				// STATIC sensitive input: the mock does not resolve STATIC inputs from the BBD.
				statecheck.ExpectKnownValue(bbAddr,
					tfjsonpath.New("all_inputs").AtMapKey("static_secret").AtMapKey("sensitive").AtMapKey("secret_hash"),
					xknownvalue.NotEmptyString()),
				// A succeeded run proves the runner cloned the fixture and ran OpenTofu.
				statecheck.ExpectKnownValue(bbAddr,
					tfjsonpath.New("status").AtMapKey("succeeded"),
					knownvalue.Bool(true)),
			)
		}

		ApplyAndTest(t, resource.TestCase{
			Steps: []resource.TestStep{
				{
					// Step 1: create BBD v1 + BB on v1. Sensitive inputs surface as hashes in all_inputs.
					Config: hclStep(t, "resources/building_block/06_sensitive_inputs_and_upgrade", 1, vars),
					ConfigPlanChecks: resource.ConfigPlanChecks{
						PreApply: []plancheck.PlanCheck{
							plancheck.ExpectResourceAction(bbAddr, plancheck.ResourceActionCreate),
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
						"purge_on_delete",
						"timeouts.create",
						"timeouts.update",
						"timeouts.delete",
					},
					ImportStateIdFunc: func(s *terraform.State) (string, error) {
						rs := s.RootModule().Resources[bbAddr]
						if rs == nil {
							return "", fmt.Errorf("resource not found: %s", bbAddr)
						}
						return rs.Primary.Attributes["metadata.uuid"], nil
					},
					ResourceName: bbAddr,
				},
				{
					// Step 3: re-draft the BBD → v2 draft. version_latest_release
					// still resolves to v1, so the BB plan is a no-op.
					Config: hclStep(t, "resources/building_block/06_sensitive_inputs_and_upgrade", 3, vars),
					ConfigPlanChecks: resource.ConfigPlanChecks{
						PreApply: []plancheck.PlanCheck{
							plancheck.ExpectResourceAction(bbdAddr, plancheck.ResourceActionUpdate),
							plancheck.ExpectResourceAction(bbAddr, plancheck.ResourceActionNoop),
						},
					},
				},
				{
					// Step 4: release BBD v2 + upgrade the BB to v2 in one apply. The
					// sensitive api_key is echoed as its hash sentinel and the secret must survive.
					Config: hclStep(t, "resources/building_block/06_sensitive_inputs_and_upgrade", 4, vars),
					ConfigPlanChecks: resource.ConfigPlanChecks{
						PreApply: []plancheck.PlanCheck{
							plancheck.ExpectResourceAction(bbdAddr, plancheck.ResourceActionUpdate),
							plancheck.ExpectResourceAction(bbAddr, plancheck.ResourceActionUpdate),
						},
					},
					ConfigStateChecks: []statecheck.StateCheck{
						statecheck.ExpectKnownValue(bbAddr, tfjsonpath.New("metadata").AtMapKey("uuid"), xknownvalue.NotEmptyString()),
						// Sensitive hash still present after upgrade (secret not corrupted).
						statecheck.ExpectKnownValue(bbAddr,
							tfjsonpath.New("all_inputs").AtMapKey("api_key").AtMapKey("sensitive").AtMapKey("secret_hash"),
							xknownvalue.NotEmptyString()),
						// Re-capture the run uuid so the rotation step below compares against the upgrade run.
						statecheck.ExpectKnownValue(bbAddr, tfjsonpath.New("status").AtMapKey("latest_run_uuid"), captureRunUuid),
					},
				},
				{
					// Step 5: post-upgrade plan must be empty — no spurious rerun and
					// no phantom-input drift.
					Config:   hclStep(t, "resources/building_block/06_sensitive_inputs_and_upgrade", 5, vars),
					PlanOnly: true,
				},
				{
					// Step 6: rotate api_key (bumped secret_version). Rotation is invisible to the rerun
					// predicate and must be detected via the changed secret_version, so this is an
					// in-place Update and latest_run_uuid must change vs. the previous run.
					Config: hclStep(t, "resources/building_block/06_sensitive_inputs_and_upgrade", 6, vars),
					ConfigPlanChecks: resource.ConfigPlanChecks{
						PreApply: []plancheck.PlanCheck{
							plancheck.ExpectResourceAction(bbAddr, plancheck.ResourceActionUpdate),
						},
					},
					ConfigStateChecks: []statecheck.StateCheck{
						statecheck.ExpectKnownValue(bbAddr,
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

	// 08_validation_rejects: a target_ref whose kind and identifier disagree is rejected by a provider-side validator.
	t.Run("08_validation_rejects", func(t *testing.T) {
		vars := hclStepVars{"suffix": acctest.RandString(8)}

		ApplyAndTest(t, resource.TestCase{
			Steps: []resource.TestStep{
				{
					// target_ref kind=meshTenant must use uuid, not name.
					Config:      hclStep(t, "resources/building_block/08_validation_rejects", 1, vars),
					ExpectError: expectErrorWords("not", "set", "kind"),
				},
				{
					// target_ref kind=meshWorkspace must use name, not uuid.
					Config:      hclStep(t, "resources/building_block/08_validation_rejects", 2, vars),
					ExpectError: expectErrorWords("not", "set", "kind"),
				},
			},
		})
	})

	// 09_purge_on_delete sets purge_on_delete = true so teardown deletes the BB via DELETE /{uuid}/purge.
	// Runs in both modes (create + flag wiring + purge teardown succeeds); the CheckDestroy below carries
	// the mechanism and its acceptance-only 404 assertion.
	t.Run("09_purge_on_delete", func(t *testing.T) {
		vars := hclStepVars{"suffix": acctest.RandString(8)}
		buildingBlockAddr := "meshstack_building_block.example_workspace"

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
					Config: hclStep(t, "resources/building_block/09_purge_on_delete", 1, vars),
					ConfigPlanChecks: resource.ConfigPlanChecks{
						PreApply: []plancheck.PlanCheck{
							plancheck.ExpectResourceAction(buildingBlockAddr, plancheck.ResourceActionCreate),
						},
					},
					ConfigStateChecks: []statecheck.StateCheck{
						statecheck.ExpectKnownValue(buildingBlockAddr, tfjsonpath.New("metadata").AtMapKey("uuid"), xknownvalue.NotEmptyString(func(v string) error {
							bbUuid = v
							return nil
						})),
						statecheck.ExpectKnownValue(buildingBlockAddr, tfjsonpath.New("purge_on_delete"), knownvalue.Bool(true)),
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
		// BBD whose `size` is a PLATFORM_OPERATOR_MANUAL_INPUT (name/environment are user inputs).
		// Create with all inputs declared (admin sets the operator input on create).
		// Then manage ONLY the operator input — drop the user inputs from the configuration.
		vars := hclStepVars{"suffix": acctest.RandString(8)}
		buildingBlockAddr := "meshstack_building_block.example_workspace"

		ApplyAndTest(t, resource.TestCase{
			Steps: []resource.TestStep{
				{
					Config:            hclStep(t, "resources/building_block/10_partial_input_ownership", 1, vars),
					ConfigStateChecks: bbv3StateChecks(buildingBlockAddr, "my-workspace-building-block", bbv3SizeEnvInputChecks(buildingBlockAddr)...),
				},
				{
					// Dropping the user inputs is an in-place Update, never a Replace.
					Config: hclStep(t, "resources/building_block/10_partial_input_ownership", 2, vars),
					ConfigPlanChecks: resource.ConfigPlanChecks{
						PreApply: []plancheck.PlanCheck{
							plancheck.ExpectResourceAction(buildingBlockAddr, plancheck.ResourceActionUpdate),
						},
					},
					ConfigStateChecks: []statecheck.StateCheck{
						// The operator input stays in spec.inputs (it is declared).
						statecheck.ExpectKnownValue(buildingBlockAddr, tfjsonpath.New("spec").AtMapKey("inputs").AtMapKey("size").AtMapKey("value"), knownvalue.StringExact("16")),
						// The un-declared user inputs are preserved, surfaced read-only in all_inputs.
						statecheck.ExpectKnownValue(buildingBlockAddr, tfjsonpath.New("all_inputs").AtMapKey("name").AtMapKey("value"), knownvalue.StringExact(`"my-name"`)),
						statecheck.ExpectKnownValue(buildingBlockAddr, tfjsonpath.New("all_inputs").AtMapKey("environment").AtMapKey("value"), knownvalue.StringExact(`"dev"`)),
						statecheck.ExpectKnownValue(buildingBlockAddr, tfjsonpath.New("all_inputs").AtMapKey("size").AtMapKey("value"), knownvalue.StringExact("16")),
					},
				},
				{
					// No drift: the omitted user inputs must not reappear as a pending change.
					Config:   hclStep(t, "resources/building_block/10_partial_input_ownership", 3, vars),
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
			// Workspace W owns the BBD; workspace O consumes it across the workspace boundary.
			// The `broken` branch fails `tofu apply` on a precondition.
			// PURGE deletion: a normal DELETE by the consumer's workspace key soft-deletes the block
			// internally without a runner destroy run. The default DELETE mode would run the `broken`
			// module again at teardown, which re-fails the precondition and leaves the block stuck in
			// FAILED — a nondeterministic post-test-destroy failure under the parallel suite. This test
			// only asserts on the failed CREATE run's transparency, so the teardown mode is immaterial.
			// Workspace-scoped key in the consumer workspace: BUILDINGBLOCK_* only, deliberately no
			// MANAGED_/ADM_ authority. Reading the failed run's logs must hinge purely on run transparency.
			// The consumer creates the BB via the meshstack-other provider (its workspace-scoped key).
			// Keep the key alive until after the BB is destroyed: the other provider needs it for teardown.
			vars := hclStepVars{"suffix": acctest.RandString(8), "repository_url": terraformTestdataRepoURL(t), "run_transparency": fmt.Sprint(runTransparency)}
			apiKeyAddr := "meshstack_api_key.reader"
			buildingBlockAddr := "meshstack_building_block.sensitive_user_input"

			var apiKeyClientId, apiKeyClientSecret lazyVariable
			consumerVariables := tfconfig.Variables{
				"apikey_client_id":     &apiKeyClientId,
				"apikey_client_secret": &apiKeyClientSecret,
			}
			// The support file's postcondition rejects the failed run the broken ref produces. With run
			// transparency ON the repair runs and fails on the broken ref again, so the postcondition fails
			// that apply too; with it OFF meshStack refuses the trigger-run and that refusal is the error.
			postconditionFailed := expectErrorWords("succeed", "health", "status")
			repairApplyErr := postconditionFailed
			if expectRepairErr != nil {
				repairApplyErr = expectRepairErr
			}

			steps := []resource.TestStep{
				{
					// Admin mints the infra + the workspace-scoped key; capture its credentials.
					Config: hclStep(t, "resources/building_block/11_run_transparency_failed_run", 1, vars),
					ConfigStateChecks: []statecheck.StateCheck{
						statecheck.ExpectKnownValue(apiKeyAddr, tfjsonpath.New("status").AtMapKey("client_id"), xknownvalue.NotEmptyString(func(clientId string) error {
							apiKeyClientId = lazyVariable(clientId)
							return nil
						})),
						statecheck.ExpectKnownValue(apiKeyAddr, tfjsonpath.New("status").AtMapKey("client_secret"), xknownvalue.NotEmptyString(func(clientSecret string) error {
							apiKeyClientSecret = lazyVariable(clientSecret)
							return nil
						})),
					},
				},
				{
					// The run fails on the broken ref, and only the postcondition turns its warning into a failed apply.
					Config:          hclStep(t, "resources/building_block/11_run_transparency_failed_run", 2, vars),
					ConfigVariables: consumerVariables,
					ConfigPlanChecks: resource.ConfigPlanChecks{
						PreApply: []plancheck.PlanCheck{
							plancheck.ExpectResourceAction(buildingBlockAddr, plancheck.ResourceActionCreate),
						},
					},
					ExpectError: postconditionFailed,
				},
				{
					// Re-applying the unchanged config is the repair. The plan check is the assertion this
					// issue is about: the create failed its apply, and the building block is still an
					// in-place update rather than a replace, so nothing it created is about to be destroyed.
					Config:          hclStep(t, "resources/building_block/11_run_transparency_failed_run", 3, vars),
					ConfigVariables: consumerVariables,
					ConfigPlanChecks: resource.ConfigPlanChecks{
						PreApply: []plancheck.PlanCheck{
							plancheck.ExpectResourceAction(buildingBlockAddr, plancheck.ResourceActionUpdate),
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
					Config:          hclStep(t, "resources/building_block/11_run_transparency_failed_run", 4, vars),
					ConfigVariables: consumerVariables,
					ConfigPlanChecks: resource.ConfigPlanChecks{
						PostApplyPostRefresh: []plancheck.PlanCheck{
							plancheck.ExpectResourceAction(buildingBlockAddr, plancheck.ResourceActionUpdate),
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
		// definition, so the repair cannot start at all, and the error links the block in meshPanel.
		t.Run("transparency_off_refuses_the_repair", func(t *testing.T) {
			runCase(t, false, expectErrorWords("refused", "run", "again", "Open", "meshPanel:", "http"))
		})
	})

	// 13_parent_child is the only scenario with a real parent. Every other parent step uses a
	// synthetic uuid and therefore never reaches a live backend.
	t.Run("13_parent_child", func(t *testing.T) {
		vars := hclStepVars{"suffix": acctest.RandString(8)}
		parentAddr := "meshstack_building_block.parent"
		childAddr := "meshstack_building_block.child"

		ApplyAndTest(t, resource.TestCase{
			Steps: []resource.TestStep{
				{
					Config: hclStep(t, "resources/building_block/13_parent_child", 1, vars),
					ConfigPlanChecks: resource.ConfigPlanChecks{
						PreApply: []plancheck.PlanCheck{
							plancheck.ExpectResourceAction(parentAddr, plancheck.ResourceActionCreate),
							plancheck.ExpectResourceAction(childAddr, plancheck.ResourceActionCreate),
						},
					},
					ConfigStateChecks: append(
						bbv3StateChecks(childAddr, "my-child-building-block"),
						statecheck.ExpectKnownValue(childAddr, tfjsonpath.New("spec").AtMapKey("parent_building_block_refs"), knownvalue.SetSizeExact(1)),
						statecheck.CompareValuePairs(
							parentAddr, tfjsonpath.New("ref"),
							childAddr, tfjsonpath.New("spec").AtMapKey("parent_building_block_refs").AtSliceIndex(0),
							compare.ValuesSame(),
						),
					),
				},
				{
					// The parent ref read back must hash to the same set element as the one the
					// configuration declares, or this plan is not empty.
					Config:   hclStep(t, "resources/building_block/13_parent_child", 2, vars),
					PlanOnly: true,
				},
			},
		})
	})

	t.Run("14_parent_change_on_drafts", func(t *testing.T) {
		vars := hclStepVars{"suffix": acctest.RandString(8)}
		otherParentAddr := "meshstack_building_block.other_parent"
		childAddr := "meshstack_building_block.child"
		ApplyAndTest(t, resource.TestCase{
			Steps: []resource.TestStep{
				{Config: hclStep(t, "resources/building_block/14_parent_change_on_drafts", 1, vars)},
				{
					Config: hclStep(t, "resources/building_block/14_parent_change_on_drafts", 2, vars),
					ConfigPlanChecks: resource.ConfigPlanChecks{
						PreApply: []plancheck.PlanCheck{plancheck.ExpectResourceAction(childAddr, plancheck.ResourceActionUpdate)},
					},
					ConfigStateChecks: []statecheck.StateCheck{statecheck.CompareValuePairs(
						otherParentAddr, tfjsonpath.New("ref"),
						childAddr, tfjsonpath.New("spec").AtMapKey("parent_building_block_refs").AtSliceIndex(0),
						compare.ValuesSame(),
					)},
				},
			},
		})
	})

	t.Run("15_delete_after_a_failed_run", func(t *testing.T) {
		if IsMockClientTest() {
			t.Skip("the mock neither fails a run nor runs a delete")
		}

		repoURL := terraformTestdataRepoURL(t)
		vars := hclStepVars{"suffix": acctest.RandString(8), "repository_url": repoURL}
		buildingBlockAddr := "meshstack_building_block.sensitive_user_input"

		ApplyAndTest(t, resource.TestCase{
			Steps: []resource.TestStep{
				{
					Config:      hclStep(t, "resources/building_block/15_delete_after_a_failed_run", 1, vars),
					ExpectError: expectErrorWords("succeed", "health", "status"),
				},
				{
					// The delete run gets a module that destroys cleanly, so only the failed run before
					// could fail this destroy.
					PreConfig: func() {
						out, err := exec.Command("git", "-C", strings.TrimPrefix(repoURL, "file://"), "branch", "-f", "broken", "main").CombinedOutput()
						require.NoError(t, err, "%s", out)
					},
					Config: hclStep(t, "resources/building_block/15_delete_after_a_failed_run", 2, vars),
					ConfigPlanChecks: resource.ConfigPlanChecks{
						PreApply: []plancheck.PlanCheck{
							plancheck.ExpectResourceAction(buildingBlockAddr, plancheck.ResourceActionDestroy),
						},
					},
				},
			},
		})
	})

	t.Run("16_hash_version_change_on_a_draft", func(t *testing.T) {
		olderHashVersion := BuildingBlockDefinitionVersionContentHash{hashVersion: currentHashVersion - 1, hashValue: "aaa"}.toBase64()
		vars := hclStepVars{"suffix": acctest.RandString(8), "content_hash": olderHashVersion}
		buildingBlockAddr := "meshstack_building_block.example_workspace"

		latestRun := statecheck.CompareValue(compare.ValuesDiffer())
		latestRunPath := tfjsonpath.New("status").AtMapKey("latest_run_uuid")
		ApplyAndTest(t, resource.TestCase{
			Steps: []resource.TestStep{
				{
					Config:            hclStep(t, "resources/building_block/16_hash_version_change_on_a_draft", 1, vars),
					ConfigStateChecks: []statecheck.StateCheck{latestRun.AddStateValue(buildingBlockAddr, latestRunPath)},
				},
				{
					Config: hclStep(t, "resources/building_block/16_hash_version_change_on_a_draft", 2, vars),
					ConfigPlanChecks: resource.ConfigPlanChecks{
						PreApply: []plancheck.PlanCheck{
							plancheck.ExpectResourceAction(buildingBlockAddr, plancheck.ResourceActionUpdate),
							plancheck.ExpectUnknownValue(buildingBlockAddr, latestRunPath),
						},
					},
					ConfigStateChecks: []statecheck.StateCheck{
						latestRun.AddStateValue(buildingBlockAddr, latestRunPath),
						statecheck.ExpectKnownValue(buildingBlockAddr, tfjsonpath.New("status").AtMapKey("succeeded"), knownvalue.Bool(true)),
					},
				},
			},
		})
	})

	// 17_upgrade_without_a_run: with skip_run_on, a version upgrade known at plan time plans no run, and the
	// apply saves the new version without waiting and locks the block, whose next update plans no run either.
	// The block is held on v1 while v2 is released, so that the upgrade step reads a known version.
	t.Run("17_upgrade_without_a_run", func(t *testing.T) {
		workspaceConfig, workspaceAddr := testconfig.Workspace(t)
		paymentMethodConfig, setPaymentMethodInput := testconfig.PaymentMethodInput(t, workspaceAddr)

		var definitionAddr testconfig.Traversal
		definition := func(draft bool) testconfig.Config {
			return testconfig.Resource{Name: "building_block", Suffix: "_01_workspace"}.TestSupportConfig(t, "").WithFirstBlock(
				testconfig.ExtractAddress(&definitionAddr),
				testconfig.OwnedByWorkspace(workspaceAddr),
				testconfig.Descend("version_spec", "draft")(testconfig.SetBool(draft)),
			)
		}

		var buildingBlockAddr testconfig.Traversal
		buildingBlock := func(definitionConfig testconfig.Config, consumers ...testconfig.ExpressionConsumer) testconfig.Config {
			return testconfig.Resource{Name: "building_block", Suffix: "_01_workspace"}.Config(t).WithFirstBlock(append([]testconfig.ExpressionConsumer{
				testconfig.ExtractAddress(&buildingBlockAddr),
				testconfig.Descend("spec", "building_block_definition_version_ref")(testconfig.SetRawExpr(`{ uuid = %s }`, definitionAddr.Join("version_latest_release", "uuid"))),
				testconfig.Descend("spec", "target_ref")(testconfig.SetAddr(workspaceAddr, "ref")),
				setPaymentMethodInput,
			}, consumers...)...).Join(workspaceConfig, paymentMethodConfig, definitionConfig)
		}
		v1Config := buildingBlock(definition(false))
		redraftConfig := buildingBlock(definition(true))
		heldOnV1Config := buildingBlock(definition(false),
			testconfig.Descend("spec", "building_block_definition_version_ref")(testconfig.SetRawExpr(`{ uuid = var.v1_uuid }`)),
		).WithRawBlock(`variable "v1_uuid" { type = string }`)
		upgradeConfig := buildingBlock(definition(false), testconfig.Descend("skip_run_on")(testconfig.SetRawExpr(`["VERSION_UPGRADE"]`)))
		lockedUpdateConfig := buildingBlock(definition(false), testconfig.Descend("spec", "inputs", "name", "value")(testconfig.SetRawExpr(`jsonencode("renamed")`)))

		var v1Uuid lazyVariable
		versionPath := tfjsonpath.New("spec").AtMapKey("building_block_definition_version_ref").AtMapKey("uuid")
		latestRunPath := tfjsonpath.New("status").AtMapKey("latest_run_uuid")
		actionPath := tfjsonpath.New("status").AtMapKey("action")
		latestRun := statecheck.CompareValue(compare.ValuesSame())
		ApplyAndTest(t, resource.TestCase{
			Steps: []resource.TestStep{
				{
					Config: v1Config.String(),
					ConfigStateChecks: []statecheck.StateCheck{
						statecheck.ExpectKnownValue(buildingBlockAddr.String(), versionPath, xknownvalue.NotEmptyString(func(uuid string) error {
							v1Uuid = lazyVariable(uuid)
							return nil
						})),
						latestRun.AddStateValue(buildingBlockAddr.String(), latestRunPath),
					},
				},
				{
					Config: redraftConfig.String(),
					ConfigPlanChecks: resource.ConfigPlanChecks{
						PreApply: []plancheck.PlanCheck{plancheck.ExpectResourceAction(buildingBlockAddr.String(), plancheck.ResourceActionNoop)},
					},
				},
				{
					Config:          heldOnV1Config.String(),
					ConfigVariables: tfconfig.Variables{"v1_uuid": &v1Uuid},
					ConfigPlanChecks: resource.ConfigPlanChecks{
						PreApply: []plancheck.PlanCheck{plancheck.ExpectResourceAction(buildingBlockAddr.String(), plancheck.ResourceActionNoop)},
					},
				},
				{
					Config: upgradeConfig.String(),
					ConfigPlanChecks: resource.ConfigPlanChecks{
						PreApply: []plancheck.PlanCheck{
							plancheck.ExpectResourceAction(buildingBlockAddr.String(), plancheck.ResourceActionUpdate),
							plancheck.ExpectKnownValue(buildingBlockAddr.String(), latestRunPath, xknownvalue.NotEmptyString()),
						},
					},
					ConfigStateChecks: []statecheck.StateCheck{
						statecheck.ExpectKnownValue(buildingBlockAddr.String(), versionPath, xknownvalue.NotEmptyString(func(uuid string) error {
							if uuid == string(v1Uuid) {
								return fmt.Errorf("expected the upgrade to v2, but the block is still on v1 %q", uuid)
							}
							return nil
						})),
						latestRun.AddStateValue(buildingBlockAddr.String(), latestRunPath),
					},
				},
				{
					Config:   upgradeConfig.String(),
					PlanOnly: true,
				},
				{
					Config: lockedUpdateConfig.String(),
					ConfigPlanChecks: resource.ConfigPlanChecks{
						PreApply: []plancheck.PlanCheck{
							plancheck.ExpectResourceAction(buildingBlockAddr.String(), plancheck.ResourceActionUpdate),
							plancheck.ExpectKnownValue(buildingBlockAddr.String(), actionPath, knownvalue.StringExact(client.BuildingBlockActionUnlockRuns.String())),
						},
					},
					ConfigStateChecks: []statecheck.StateCheck{
						statecheck.ExpectKnownValue(buildingBlockAddr.String(), actionPath, knownvalue.StringExact(client.BuildingBlockActionUnlockRuns.String())),
						latestRun.AddStateValue(buildingBlockAddr.String(), latestRunPath),
					},
				},
			},
		})
	})
}

// bbv3StateChecks returns the baseline state checks shared by every BB v3 create and move step.
func bbv3StateChecks(buildingBlockAddr string, displayName string, extra ...statecheck.StateCheck) []statecheck.StateCheck {
	checks := []statecheck.StateCheck{
		statecheck.ExpectKnownValue(buildingBlockAddr, tfjsonpath.New("metadata").AtMapKey("uuid"), xknownvalue.NotEmptyString()),
		xknownvalue.Ref(buildingBlockAddr, client.MeshObjectKind.BuildingBlock, nil),
		statecheck.ExpectKnownValue(buildingBlockAddr, tfjsonpath.New("spec").AtMapKey("display_name"), knownvalue.StringExact(displayName)),
		statecheck.ExpectKnownValue(buildingBlockAddr, tfjsonpath.New("spec").AtMapKey("inputs").AtMapKey("name").AtMapKey("value"), knownvalue.StringExact(`"my-name"`)),
		statecheck.ExpectKnownValue(buildingBlockAddr, tfjsonpath.New("status").AtMapKey("health_status"), xknownvalue.NotEmptyString()),
		statecheck.ExpectKnownValue(buildingBlockAddr, tfjsonpath.New("status").AtMapKey("action"), xknownvalue.NotEmptyString()),
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

func bbv3PaymentMethodInputChecks(buildingBlockAddr string, suffix string) []statecheck.StateCheck {
	checks := []statecheck.StateCheck{
		statecheck.ExpectKnownValue(buildingBlockAddr, tfjsonpath.New("spec").AtMapKey("inputs").AtMapKey("payment_method").AtMapKey("value"),
			knownvalue.StringExact(`{"kind":"meshPaymentMethod","name":"test-pm-`+suffix+`"}`)),
	}
	// The mock reports every consumer input as USER_INPUT.
	if !IsMockClientTest() {
		checks = append(checks, statecheck.ExpectKnownValue(buildingBlockAddr, tfjsonpath.New("all_inputs").AtMapKey("payment_method").AtMapKey("assignment_type"),
			knownvalue.StringExact(client.MeshBuildingBlockInputAssignmentTypePaymentMethod.String())))
	}
	return checks
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
			require.Equal(t, tt.want, rerunNeeded(tt.plan, tt.state, false))
		})
	}
}
