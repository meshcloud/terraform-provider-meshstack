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

	testconfig "github.com/meshcloud/terraform-provider-meshstack/internal/provider/acctest/testconfig"
	xknownvalue "github.com/meshcloud/terraform-provider-meshstack/internal/provider/acctest/xknownvalue"
)

// resolvedSubject asserts that meshStack filled every placeholder of the runner's subject template in.
func resolvedSubject(prefix string) knownvalue.Check {
	return xknownvalue.NotEmptyString(func(actualValue string) error {
		if !strings.HasPrefix(actualValue, prefix) {
			return fmt.Errorf("expected the resolved subject to start with %q, got %q", prefix, actualValue)
		}
		if strings.Contains(actualValue, "{{") {
			return fmt.Errorf("expected a resolved subject, but %q still contains a placeholder", actualValue)
		}
		return nil
	})
}

// resolvedWif is the identity runs of a version present: the runner's issuer, its subject template filled
// in for the definition, and the runner's GCP block.
func resolvedWif(issuer string, subject knownvalue.Check, gcp knownvalue.Check) knownvalue.Check {
	return xknownvalue.MapExact(map[string]knownvalue.Check{
		"issuer":  knownvalue.StringExact(issuer),
		"subject": subject,
		"gcp":     gcp,
		"aws":     knownvalue.Null(),
		"azure":   knownvalue.Null(),
	})
}

// expectedStatus is the status of a definition whose versions, the given numbers, all run on one runner:
// every entry carries wif, and so does the top-level block, which is the identity of the highest version.
func expectedStatus(wif knownvalue.Check, numbers ...int64) knownvalue.Check {
	versions := make([]knownvalue.Check, len(numbers))
	for i, number := range numbers {
		versions[i] = xknownvalue.MapExact(map[string]knownvalue.Check{
			"uuid":                         knownvalue.NotNull(),
			"number":                       knownvalue.Int64Exact(number),
			"workload_identity_federation": wif,
		})
	}
	return xknownvalue.MapExact(map[string]knownvalue.Check{
		"versions":                     knownvalue.ListExact(versions),
		"workload_identity_federation": wif,
	})
}

func TestAccBuildingBlockDefinitionWifStatus(t *testing.T) {
	config, bbdAddr, _, otherRunnerAddr := testconfig.BBDTerraformWithWifRunners(t, runnerPublicKey)
	statusPath := tfjsonpath.New("status")
	subjectPath := statusPath.AtMapKey("workload_identity_federation").AtMapKey("subject")

	gcp := xknownvalue.MapExact(map[string]knownvalue.Check{
		"audience":   knownvalue.StringExact("gcp-workload-identity-provider:namespace"),
		"token_path": knownvalue.StringExact("/var/run/secrets/workload-identity/token"),
	})
	otherRunnerSubject := resolvedSubject("system:serviceaccount:other-namespace:bbd.")
	otherRunnerWif := resolvedWif(testconfig.OtherWifRunnerIssuer, otherRunnerSubject, gcp)

	// meshStack stores a version's secrets encrypted for its runner, so moving the definition to another
	// one has to re-supply every secret as plaintext - a hash is a 400. Bumping secret_version is how the
	// provider is asked for that, same as the rotation step of TestAccBuildingBlockDefinition.
	const movedToOtherRunner = "moved-to-other-runner"
	otherRunnerConfig := config.WithFirstBlock(
		testconfig.Descend("version_spec", "runner_ref")(testconfig.SetAddr(otherRunnerAddr, "ref")),
		testconfig.Descend("version_spec", "implementation", "terraform", "ssh_private_key", "secret_version")(
			testconfig.SetString(movedToOtherRunner),
		),
		testconfig.Descend("version_spec", "inputs", "SOMETHING_VERY_SECRET", "sensitive", "argument", "secret_version")(
			testconfig.SetString(movedToOtherRunner),
		),
	)
	renamedConfig := otherRunnerConfig.WithFirstBlock(
		testconfig.Descend("spec", "display_name")(testconfig.SetString("Example Building Block, renamed")),
	)

	ApplyAndTest(t, resource.TestCase{
		Steps: []resource.TestStep{
			{
				Config: config.String(),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction(bbdAddr.String(), plancheck.ResourceActionCreate),
					},
				},
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(bbdAddr.String(), statusPath,
						expectedStatus(resolvedWif("https://oidc.example.com", resolvedSubject("system:serviceaccount:namespace:workspace."), gcp), 1)),
				},
			},
			{
				// A changed runner changes the identity of the version it is changed on, so status is read again.
				Config: otherRunnerConfig.String(),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction(bbdAddr.String(), plancheck.ResourceActionUpdate),
						plancheck.ExpectUnknownValue(bbdAddr.String(), statusPath),
					},
				},
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(bbdAddr.String(), statusPath, expectedStatus(otherRunnerWif, 1)),
				},
			},
			{
				// An unrelated change keeps it, instead of planning a pointless "known after apply".
				Config: renamedConfig.String(),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction(bbdAddr.String(), plancheck.ResourceActionUpdate),
						plancheck.ExpectKnownValue(bbdAddr.String(), subjectPath, otherRunnerSubject),
					},
				},
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(bbdAddr.String(), statusPath, expectedStatus(otherRunnerWif, 1)),
				},
			},
			{
				// A release changes the version in place, so the identity of the version stays as it was.
				Config: releaseBBDVersion(t, renamedConfig).String(),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction(bbdAddr.String(), plancheck.ResourceActionUpdate),
						plancheck.ExpectKnownValue(bbdAddr.String(), subjectPath, otherRunnerSubject),
					},
				},
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(bbdAddr.String(), statusPath, expectedStatus(otherRunnerWif, 1)),
				},
			},
			{
				// A new draft cut from the released version adds an entry, so status is read again.
				Config: updateBBDDescription(t, renamedConfig, "An updated building block definition").String(),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction(bbdAddr.String(), plancheck.ResourceActionUpdate),
						plancheck.ExpectUnknownValue(bbdAddr.String(), statusPath),
					},
				},
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(bbdAddr.String(), statusPath, expectedStatus(otherRunnerWif, 1, 2)),
				},
			},
		},
	})
}
