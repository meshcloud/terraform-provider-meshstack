package provider

import (
	"fmt"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/knownvalue"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
	"github.com/hashicorp/terraform-plugin-testing/statecheck"
	"github.com/hashicorp/terraform-plugin-testing/tfjsonpath"

	"github.com/meshcloud/terraform-provider-meshstack/internal/provider/acctest/testconfig"
	"github.com/meshcloud/terraform-provider-meshstack/internal/provider/acctest/xknownvalue"
)

// assertIsHashNotPlaintext validates that a surfaced sensitive-input value is the backend's secret
// hash and not the leaked plaintext. It guards against the toResourceModel fallback that stuffs a
// non-string value's raw Go representation (e.g. "map[plaintext:...]") into value_string when the
// secret was demoted from SecretOrAny.X to Y (IsSensitive not set on the outbound DTO).
func assertIsHashNotPlaintext(plaintext string) func(string) error {
	return func(v string) error {
		// The mock's simulated hash is "sha256:<plaintext>", so a substring match on the plaintext
		// would false-positive there; the meaningful guards are that the value is not the raw
		// plaintext and not the toResourceModel map-fallback representation.
		if v == plaintext {
			return fmt.Errorf("expected a secret hash but got the raw plaintext %q — the secret was not hashed", v)
		}
		if strings.Contains(v, "map[") {
			return fmt.Errorf("expected a secret hash but got %q — the plaintext leaked (secret demoted to a plain value)", v)
		}
		return nil
	}
}

func TestAccBuildingBlockV2(t *testing.T) {
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
		if IsMockClientTest() {
			// The in-memory mock does not resolve STATIC inputs from the BBD, so the
			// static secret never appears in combined_inputs in mock mode.
			t.Skip("requires real meshStack to resolve static secret inputs")
		}

		workspaceConfig, workspaceAddr := testconfig.Workspace(t)
		exampleResource := testconfig.Resource{Name: "building_block_v2", Suffix: "_03_sensitive_input"}

		var buildingBlockDefinitionAddr testconfig.Traversal
		buildingBlockDefinitionConfig := exampleResource.TestSupportConfig(t, "_bbd").WithFirstBlock(
			testconfig.ExtractAddress(&buildingBlockDefinitionAddr),
			testconfig.OwnedByWorkspace(workspaceAddr),
			// Point at the committed bare repo served over loopback so the run actually completes and the
			// block reaches a final state, letting the default wait_for_completion/purge_on_delete exercise
			// the full lifecycle instead of leaving a stuck run behind.
			testconfig.Descend("version_spec", "implementation", "terraform", "repository_url")(
				testconfig.SetRawExpr("%q", terraformTestdataRepoURL(t)),
			),
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
	})

	t.Run("04_sensitive_user_input", func(t *testing.T) {
		// Runs in both modes. Sensitive USER_INPUTs (STRING and CODE) are sent as SecretEmbedded
		// {"plaintext": "..."} with IsSensitive=true, which keeps them in the SecretOrAny.X variant
		// across a JSON round-trip; the backend (and the mock's backendSecretBehavior) return only the
		// sha256 hash, which surfaces in combined_inputs (STRING hash in value_string, CODE hash in
		// value_code). STRING and CODE take the identical code path — the only difference is which
		// value_* field the hash lands in. The assertions verify the surfaced value is a real hash, not
		// the leaked plaintext (a prior bug demoted the secret to a plain value and stuffed its raw map
		// representation into value_string via the toResourceModel fallback).
		workspaceConfig, workspaceAddr := testconfig.Workspace(t)
		exampleResource := testconfig.Resource{Name: "building_block_v2", Suffix: "_04_sensitive_user_input"}

		var buildingBlockDefinitionAddr testconfig.Traversal
		buildingBlockDefinitionConfig := exampleResource.TestSupportConfig(t, "_bbd").WithFirstBlock(
			testconfig.ExtractAddress(&buildingBlockDefinitionAddr),
			testconfig.OwnedByWorkspace(workspaceAddr),
			// Point at the committed bare repo served over loopback so the run actually completes and the
			// block reaches a final state, letting the default wait_for_completion/purge_on_delete exercise
			// the full lifecycle instead of leaving a stuck run behind.
			testconfig.Descend("version_spec", "implementation", "terraform", "repository_url")(
				testconfig.SetRawExpr("%q", terraformTestdataRepoURL(t)),
			),
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
							xknownvalue.NotEmptyString(assertIsHashNotPlaintext("super-secret-string-value"))),
						statecheck.ExpectKnownValue(buildingBlockAddr.String(),
							tfjsonpath.New("spec").AtMapKey("combined_inputs").AtMapKey("secret_code").AtMapKey("value_code"),
							xknownvalue.NotEmptyString(assertIsHashNotPlaintext("super-secret-code-value"))),
					},
				},
			},
		})
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
