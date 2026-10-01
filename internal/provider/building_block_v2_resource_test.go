package provider

import (
	"fmt"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/knownvalue"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
	"github.com/hashicorp/terraform-plugin-testing/statecheck"
	"github.com/hashicorp/terraform-plugin-testing/tfjsonpath"

	"github.com/meshcloud/terraform-provider-meshstack/examples"
	"github.com/meshcloud/terraform-provider-meshstack/internal/provider/acctest/xknownvalue"
)

// Addresses of the blocks in examples/resources/meshstack_building_block_v2/resource-test-*.tf.
const (
	bbv2WorkspaceAddr          = "meshstack_building_block_v2.example_workspace"
	bbv2TenantAddr             = "meshstack_building_block_v2.example_tenant"
	bbv2SensitiveAddr          = "meshstack_building_block_v2.sensitive"
	bbv2SensitiveUserInputAddr = "meshstack_building_block_v2.sensitive_user_input"
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
		config := examples.JoinTestStepConfigs(
			examples.Resource.TestStepConfig(t, "building_block_v2", 1, "01_workspace"),
			examples.Resource.TestStepConfig(t, "workspace", 1, "variables", "prerequisites"),
		)

		ApplyAndTest(t, resource.TestCase{
			Steps: []resource.TestStep{
				{
					Config:          config,
					ConfigVariables: NewVariablesWithSuffix(acctest.RandString(8)),
					ConfigPlanChecks: resource.ConfigPlanChecks{
						PreApply: []plancheck.PlanCheck{
							plancheck.ExpectResourceAction(bbv2WorkspaceAddr, plancheck.ResourceActionCreate),
						},
					},
					ConfigStateChecks: bbv2StateChecks(bbv2WorkspaceAddr, "my-workspace-building-block"),
				},
			},
		})
	})

	t.Run("02_tenant", func(t *testing.T) {
		config := examples.JoinTestStepConfigs(
			examples.Resource.TestStepConfig(t, "building_block_v2", 2, "02_tenant"),
			tenantStepConfig(t, 1, 8, 1),
		)

		ApplyAndTest(t, resource.TestCase{
			Steps: []resource.TestStep{
				{
					Config:          config,
					ConfigVariables: NewVariablesWithSuffix(acctest.RandString(8)),
					ConfigPlanChecks: resource.ConfigPlanChecks{
						PreApply: []plancheck.PlanCheck{
							plancheck.ExpectResourceAction(bbv2TenantAddr, plancheck.ResourceActionCreate),
						},
					},
					ConfigStateChecks: bbv2StateChecks(bbv2TenantAddr, "my-tenant-building-block"),
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

		config := examples.JoinTestStepConfigs(
			examples.Resource.TestStepConfig(t, "building_block_v2", 3, "03_sensitive_input_bbd", "variables"),
			examples.Resource.TestStepConfig(t, "workspace", 1, "variables", "prerequisites"),
		)

		ApplyAndTest(t, resource.TestCase{
			Steps: []resource.TestStep{
				{
					Config:          config,
					ConfigVariables: With(NewVariablesWithSuffix(acctest.RandString(8)), "terraform_repository_url", terraformTestdataRepoURL(t)),
					ConfigPlanChecks: resource.ConfigPlanChecks{
						PreApply: []plancheck.PlanCheck{
							plancheck.ExpectResourceAction(bbv2SensitiveAddr, plancheck.ResourceActionCreate),
						},
					},
					ConfigStateChecks: []statecheck.StateCheck{
						statecheck.ExpectKnownValue(bbv2SensitiveAddr, tfjsonpath.New("metadata").AtMapKey("uuid"), xknownvalue.NotEmptyString()),
						// The read fix surfaces the embedded-secret hash here; without it this is null.
						statecheck.ExpectKnownValue(bbv2SensitiveAddr,
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
		config := examples.JoinTestStepConfigs(
			examples.Resource.TestStepConfig(t, "building_block_v2", 4, "04_sensitive_user_input_bbd", "variables"),
			examples.Resource.TestStepConfig(t, "workspace", 1, "variables", "prerequisites"),
		)

		ApplyAndTest(t, resource.TestCase{
			Steps: []resource.TestStep{
				{
					Config:          config,
					ConfigVariables: With(NewVariablesWithSuffix(acctest.RandString(8)), "terraform_repository_url", terraformTestdataRepoURL(t)),
					ConfigPlanChecks: resource.ConfigPlanChecks{
						PreApply: []plancheck.PlanCheck{
							plancheck.ExpectResourceAction(bbv2SensitiveUserInputAddr, plancheck.ResourceActionCreate),
						},
					},
					ConfigStateChecks: []statecheck.StateCheck{
						statecheck.ExpectKnownValue(bbv2SensitiveUserInputAddr, tfjsonpath.New("metadata").AtMapKey("uuid"), xknownvalue.NotEmptyString()),
						// Sensitive user inputs are sent as {"plaintext":...}; the API returns the hash.
						// The hash surfaces in combined_inputs (the STRING hash in value_string, the CODE hash in value_code).
						statecheck.ExpectKnownValue(bbv2SensitiveUserInputAddr,
							tfjsonpath.New("spec").AtMapKey("combined_inputs").AtMapKey("secret_str").AtMapKey("value_string"),
							xknownvalue.NotEmptyString(assertIsHashNotPlaintext("super-secret-string-value"))),
						statecheck.ExpectKnownValue(bbv2SensitiveUserInputAddr,
							tfjsonpath.New("spec").AtMapKey("combined_inputs").AtMapKey("secret_code").AtMapKey("value_code"),
							xknownvalue.NotEmptyString(assertIsHashNotPlaintext("super-secret-code-value"))),
					},
				},
			},
		})
	})
}

func bbv2StateChecks(buildingBlockAddr, displayName string) []statecheck.StateCheck {
	return []statecheck.StateCheck{
		statecheck.ExpectKnownValue(buildingBlockAddr, tfjsonpath.New("metadata").AtMapKey("uuid"), xknownvalue.NotEmptyString()),
		statecheck.ExpectKnownValue(buildingBlockAddr, tfjsonpath.New("spec").AtMapKey("display_name"), knownvalue.StringExact(displayName)),
		statecheck.ExpectKnownValue(buildingBlockAddr, tfjsonpath.New("spec").AtMapKey("inputs").AtMapKey("name").AtMapKey("value_string"), knownvalue.StringExact("my-name")),
		statecheck.ExpectKnownValue(buildingBlockAddr, tfjsonpath.New("spec").AtMapKey("inputs").AtMapKey("size").AtMapKey("value_int"), knownvalue.Int64Exact(16)),
		statecheck.ExpectKnownValue(buildingBlockAddr, tfjsonpath.New("spec").AtMapKey("inputs").AtMapKey("environment").AtMapKey("value_single_select"), knownvalue.StringExact("dev")),
		statecheck.ExpectKnownValue(buildingBlockAddr, tfjsonpath.New("status").AtMapKey("status"), knownvalue.StringExact("SUCCEEDED")),
	}
}
