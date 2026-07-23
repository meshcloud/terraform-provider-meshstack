package provider

import (
	"crypto/sha256"
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/knownvalue"
	"github.com/hashicorp/terraform-plugin-testing/statecheck"
)

func TestAccNonEphemeralSecretFunction(t *testing.T) {
	for _, value := range []string{"top-secret-value", "another secret", "  spaces around  ", ""} {
		t.Run(fmt.Sprintf("%q", value), func(t *testing.T) {
			hash := fmt.Sprintf("%x", sha256.Sum256([]byte(value)))
			ApplyAndTest(t, resource.TestCase{
				Steps: []resource.TestStep{{
					Config: fmt.Sprintf(`output "secret" {
  value = provider::meshstack::non_ephemeral_secret(%q)
}`, value),
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

	// A plain output of a sensitive value fails to apply with "Output refers to sensitive values".
	// This one applies cleanly, which proves nonsensitive() really did strip the mark.
	t.Run("sensitive local wrapped in nonsensitive is not sensitive", func(t *testing.T) {
		const value = "mock-pat-token-12345"
		hash := fmt.Sprintf("%x", sha256.Sum256([]byte(value)))
		ApplyAndTest(t, resource.TestCase{
			Steps: []resource.TestStep{{
				Config: fmt.Sprintf(`locals {
  some_sensitive_value = sensitive(%q)
}
output "secret" {
  value = provider::meshstack::non_ephemeral_secret(nonsensitive(local.some_sensitive_value))
}`, value),
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
