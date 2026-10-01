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
	xknownvalue "github.com/meshcloud/terraform-provider-meshstack/internal/provider/acctest/xknownvalue"
)

// otherWifRunnerIssuer is the issuer of the runner in
// examples/resources/meshstack_building_block_runner/test-support_other-wif-runner.tf.
const otherWifRunnerIssuer = "https://oidc-other.example.com"

// bbdWifStepConfig is a terraform definition step running on one of two runners with workload identity
// federation: resource-test-23.tf picks the first, resource-test-24.tf the other.
func bbdWifStepConfig(t *testing.T, index int) string {
	t.Helper()
	return examples.JoinTestStepConfigs(
		bbdStepConfig(t, index, terraformBbdSupports...),
		examples.Resource.TestStepConfig(t, "building_block_runner", 9, "variables", "other-wif-runner"),
	)
}

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

// expectedVersions is the versions list of a definition whose versions, the given numbers, all run on
// one runner, so every entry carries the same identity.
func expectedVersions(wif knownvalue.Check, numbers ...int64) knownvalue.Check {
	versions := make([]knownvalue.Check, len(numbers))
	for i, number := range numbers {
		versions[i] = knownvalue.ObjectPartial(map[string]knownvalue.Check{
			"number":                       knownvalue.Int64Exact(number),
			"workload_identity_federation": wif,
		})
	}
	return knownvalue.ListExact(versions)
}

func TestAccBuildingBlockDefinitionWif(t *testing.T) {
	suffix := acctest.RandString(8)
	vars := With(NewVariablesWithSuffix(suffix), "tag_suffix", suffix, "runner_public_key", runnerPublicKey)
	versionsPath := tfjsonpath.New("versions")
	latestWifPath := tfjsonpath.New("version_latest").AtMapKey("workload_identity_federation")
	latestSubjectPath := latestWifPath.AtMapKey("subject")
	latestReleaseWifPath := tfjsonpath.New("version_latest_release").AtMapKey("workload_identity_federation")

	gcp := xknownvalue.MapExact(map[string]knownvalue.Check{
		"audience":   knownvalue.StringExact("gcp-workload-identity-provider:namespace"),
		"token_path": knownvalue.StringExact("/var/run/secrets/workload-identity/token"),
	})
	otherRunnerSubject := resolvedSubject("system:serviceaccount:other-namespace:bbd.")
	otherRunnerWif := resolvedWif(otherWifRunnerIssuer, otherRunnerSubject, gcp)

	renamedVars := With(vars, "display_name", "Example Building Block, renamed")
	releasedVars := With(renamedVars, "draft", false)
	redraftedVars := With(renamedVars, "description", "An updated building block definition")
	replacedVars := With(redraftedVars, "only_apply_once_per_tenant", false)

	// The subject carries the definition's uuid, so the replacement has to present a different one.
	var subjectBeforeReplacement string
	recordSubject := xknownvalue.NotEmptyString(func(actualValue string) error {
		subjectBeforeReplacement = actualValue
		return nil
	})
	replacementSubject := xknownvalue.NotEmptyString(func(actualValue string) error {
		if actualValue == subjectBeforeReplacement {
			return fmt.Errorf("expected the replacement to present its own subject, but it kept %q", actualValue)
		}
		return nil
	})

	ApplyAndTest(t, resource.TestCase{
		Steps: []resource.TestStep{
			{
				Config:          bbdWifStepConfig(t, 23),
				ConfigVariables: vars,
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction(terraformBbdAddr, plancheck.ResourceActionCreate),
					},
				},
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(terraformBbdAddr, latestWifPath,
						resolvedWif("https://oidc.example.com", resolvedSubject("system:serviceaccount:namespace:workspace."), gcp)),
					statecheck.ExpectKnownValue(terraformBbdAddr, versionsPath,
						expectedVersions(resolvedWif("https://oidc.example.com", resolvedSubject("system:serviceaccount:namespace:workspace."), gcp), 1)),
					statecheck.ExpectKnownValue(terraformBbdAddr, tfjsonpath.New("version_latest_release"), knownvalue.Null()),
				},
			},
			{
				// A changed runner changes the identity of the version it is changed on, so it is read again.
				Config:          bbdWifStepConfig(t, 24),
				ConfigVariables: vars,
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction(terraformBbdAddr, plancheck.ResourceActionUpdate),
						plancheck.ExpectUnknownValue(terraformBbdAddr, latestWifPath),
					},
				},
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(terraformBbdAddr, latestWifPath, otherRunnerWif),
					statecheck.ExpectKnownValue(terraformBbdAddr, versionsPath, expectedVersions(otherRunnerWif, 1)),
				},
			},
			{
				// An unrelated change keeps it, instead of planning a pointless "known after apply".
				Config:          bbdWifStepConfig(t, 24),
				ConfigVariables: renamedVars,
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction(terraformBbdAddr, plancheck.ResourceActionUpdate),
						plancheck.ExpectKnownValue(terraformBbdAddr, latestSubjectPath, otherRunnerSubject),
					},
				},
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(terraformBbdAddr, latestWifPath, otherRunnerWif),
				},
			},
			{
				// A release changes the version in place, so the identity of the version stays as it was and
				// becomes the identity of the latest release.
				Config:          bbdWifStepConfig(t, 24),
				ConfigVariables: releasedVars,
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction(terraformBbdAddr, plancheck.ResourceActionUpdate),
						plancheck.ExpectKnownValue(terraformBbdAddr, latestSubjectPath, otherRunnerSubject),
					},
				},
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(terraformBbdAddr, latestWifPath, otherRunnerWif),
					statecheck.ExpectKnownValue(terraformBbdAddr, latestReleaseWifPath, otherRunnerWif),
				},
			},
			{
				// A new draft cut from the released version adds an entry whose identity is read after it is
				// written, while the released version keeps its own.
				Config:          bbdWifStepConfig(t, 24),
				ConfigVariables: redraftedVars,
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction(terraformBbdAddr, plancheck.ResourceActionUpdate),
						plancheck.ExpectUnknownValue(terraformBbdAddr, latestWifPath),
						plancheck.ExpectKnownValue(terraformBbdAddr, latestReleaseWifPath, otherRunnerWif),
					},
				},
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(terraformBbdAddr, versionsPath, expectedVersions(otherRunnerWif, 1, 2)),
					statecheck.ExpectKnownValue(terraformBbdAddr, latestReleaseWifPath, otherRunnerWif),
					statecheck.ExpectKnownValue(terraformBbdAddr, latestSubjectPath, recordSubject),
				},
			},
			{
				// A replacement creates a new definition with a single version and its own subject. The create
				// side of it plans every computed attribute unknown, the version entry included.
				Config:          bbdWifStepConfig(t, 24),
				ConfigVariables: replacedVars,
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction(terraformBbdAddr, plancheck.ResourceActionReplace),
						plancheck.ExpectUnknownValue(terraformBbdAddr, tfjsonpath.New("version_latest")),
					},
				},
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(terraformBbdAddr, versionsPath,
						expectedVersions(resolvedWif(otherWifRunnerIssuer, replacementSubject, gcp), 1)),
				},
			},
		},
	})
}
