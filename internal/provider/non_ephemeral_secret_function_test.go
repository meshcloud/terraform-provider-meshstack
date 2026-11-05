package provider

import (
	"crypto/sha256"
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/knownvalue"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
	"github.com/hashicorp/terraform-plugin-testing/statecheck"
	"github.com/hashicorp/terraform-plugin-testing/tfjsonpath"

	"github.com/meshcloud/terraform-provider-meshstack/internal/provider/acctest/xknownvalue"
)

func TestAccNonEphemeralSecretFunction(t *testing.T) {
	for _, value := range []string{"top-secret-value", "another secret", "  spaces around  ", ""} {
		t.Run(fmt.Sprintf("%q", value), func(t *testing.T) {
			hash := fmt.Sprintf("%x", sha256.Sum256([]byte(value)))
			ApplyAndTest(t, resource.TestCase{
				Steps: []resource.TestStep{{
					Config: hclStep(t, "functions/non_ephemeral_secret", 1, hclStepVars{"value": value}),
					ConfigStateChecks: []statecheck.StateCheck{
						statecheck.ExpectKnownOutputValue("secret", knownvalue.ObjectExact(map[string]knownvalue.Check{
							"secret_value":   knownvalue.StringExact(value),
							"secret_version": knownvalue.StringExact(hash),
						})),
					},
				}},
			})
		})
	}

	// Guards against "returned a value for the write-only attribute ... during planning".
	t.Run("value unknown while planning", func(t *testing.T) {
		const value = "token-from-another-resource"
		hash := fmt.Sprintf("%x", sha256.Sum256([]byte(value)))
		vars := hclStepVars{"suffix": acctest.RandString(8)}
		resourceAddress := "meshstack_integration.test"

		ApplyAndTest(t, resource.TestCase{
			Steps: []resource.TestStep{{
				Config: hclStep(t, "functions/non_ephemeral_secret/value_unknown_while_planning", 1, vars),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectUnknownValue(resourceAddress, azureDevopsPatPath().AtMapKey("secret_version")),
					},
				},
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(resourceAddress,
						azureDevopsPatPath().AtMapKey("secret_version"), knownvalue.StringExact(hash)),
				},
			}},
		})
	})

	t.Run("example", func(t *testing.T) {
		hash := fmt.Sprintf("%x", sha256.Sum256([]byte("mock-pat-token-12345")))
		suffix := acctest.RandString(8)
		vars := hclStepVars{"suffix": suffix}
		resourceAddress := "meshstack_integration.example"

		ApplyAndTest(t, resource.TestCase{
			Steps: []resource.TestStep{{
				Config: hclStep(t, "functions/non_ephemeral_secret/example", 1, vars),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction(resourceAddress, plancheck.ResourceActionCreate),
						plancheck.ExpectKnownValue(resourceAddress, azureDevopsPatPath().AtMapKey("secret_version"), knownvalue.StringExact(hash)),
					},
				},
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(resourceAddress, tfjsonpath.New("metadata"), checkIntegrationMetadata("test-ws-"+suffix)),
					statecheck.ExpectKnownValue(resourceAddress, tfjsonpath.New("spec"), checkIntegrationSpec("02_azure_devops", "Azure DevOps Integration")),
					statecheck.ExpectKnownValue(resourceAddress, azureDevopsPatPath(), xknownvalue.MapExact(map[string]knownvalue.Check{
						"secret_value":   knownvalue.Null(),
						"secret_hash":    xknownvalue.NotEmptyString(),
						"secret_version": knownvalue.StringExact(hash),
					})),
				},
			}},
		})
	})

	// A plain output of a sensitive value fails to apply with "Output refers to sensitive values".
	// This one applies cleanly, which proves nonsensitive() really did strip the mark.
	t.Run("sensitive local wrapped in nonsensitive is not sensitive", func(t *testing.T) {
		const value = "mock-pat-token-12345"
		hash := fmt.Sprintf("%x", sha256.Sum256([]byte(value)))
		ApplyAndTest(t, resource.TestCase{
			Steps: []resource.TestStep{{
				Config: hclStep(t, "functions/non_ephemeral_secret/sensitive_local_wrapped_in_nonsensitive_is_not_sensitive", 1, nil),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownOutputValue("secret", knownvalue.ObjectExact(map[string]knownvalue.Check{
						"secret_value":   knownvalue.StringExact(value),
						"secret_version": knownvalue.StringExact(hash),
					})),
				},
			}},
		})
	})
}
