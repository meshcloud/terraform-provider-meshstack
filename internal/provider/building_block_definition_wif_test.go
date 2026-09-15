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
	config, bbdAddr, _, otherRunnerAddr := testconfig.BBDTerraformWithWifRunners(t, runnerPublicKey)
	versionsPath := tfjsonpath.New("versions")
	latestWifPath := tfjsonpath.New("version_latest").AtMapKey("workload_identity_federation")
	latestSubjectPath := latestWifPath.AtMapKey("subject")
	latestReleaseWifPath := tfjsonpath.New("version_latest_release").AtMapKey("workload_identity_federation")

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
	redraftedConfig := updateBBDDescription(t, renamedConfig, "An updated building block definition")
	replacedConfig := redraftedConfig.WithFirstBlock(
		testconfig.Descend("version_spec", "only_apply_once_per_tenant")(testconfig.SetBool(false)),
	)

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
				Config: config.String(),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction(bbdAddr.String(), plancheck.ResourceActionCreate),
					},
				},
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(bbdAddr.String(), latestWifPath,
						resolvedWif("https://oidc.example.com", resolvedSubject("system:serviceaccount:namespace:workspace."), gcp)),
					statecheck.ExpectKnownValue(bbdAddr.String(), versionsPath,
						expectedVersions(resolvedWif("https://oidc.example.com", resolvedSubject("system:serviceaccount:namespace:workspace."), gcp), 1)),
					statecheck.ExpectKnownValue(bbdAddr.String(), tfjsonpath.New("version_latest_release"), knownvalue.Null()),
				},
			},
			{
				// A changed runner changes the identity of the version it is changed on, so it is read again.
				Config: otherRunnerConfig.String(),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction(bbdAddr.String(), plancheck.ResourceActionUpdate),
						plancheck.ExpectUnknownValue(bbdAddr.String(), latestWifPath),
					},
				},
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(bbdAddr.String(), latestWifPath, otherRunnerWif),
					statecheck.ExpectKnownValue(bbdAddr.String(), versionsPath, expectedVersions(otherRunnerWif, 1)),
				},
			},
			{
				// An unrelated change keeps it, instead of planning a pointless "known after apply".
				Config: renamedConfig.String(),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction(bbdAddr.String(), plancheck.ResourceActionUpdate),
						plancheck.ExpectKnownValue(bbdAddr.String(), latestSubjectPath, otherRunnerSubject),
					},
				},
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(bbdAddr.String(), latestWifPath, otherRunnerWif),
				},
			},
			{
				// A release changes the version in place, so the identity of the version stays as it was and
				// becomes the identity of the latest release.
				Config: releaseBBDVersion(t, renamedConfig).String(),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction(bbdAddr.String(), plancheck.ResourceActionUpdate),
						plancheck.ExpectKnownValue(bbdAddr.String(), latestSubjectPath, otherRunnerSubject),
					},
				},
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(bbdAddr.String(), latestWifPath, otherRunnerWif),
					statecheck.ExpectKnownValue(bbdAddr.String(), latestReleaseWifPath, otherRunnerWif),
				},
			},
			{
				// A new draft cut from the released version adds an entry whose identity is read after it is
				// written, while the released version keeps its own.
				Config: redraftedConfig.String(),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction(bbdAddr.String(), plancheck.ResourceActionUpdate),
						plancheck.ExpectUnknownValue(bbdAddr.String(), latestWifPath),
						plancheck.ExpectKnownValue(bbdAddr.String(), latestReleaseWifPath, otherRunnerWif),
					},
				},
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(bbdAddr.String(), versionsPath, expectedVersions(otherRunnerWif, 1, 2)),
					statecheck.ExpectKnownValue(bbdAddr.String(), latestReleaseWifPath, otherRunnerWif),
					statecheck.ExpectKnownValue(bbdAddr.String(), latestSubjectPath, recordSubject),
				},
			},
			{
				// A replacement creates a new definition with a single version and its own subject. The create
				// side of it plans every computed attribute unknown, the version entry included.
				Config: replacedConfig.String(),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction(bbdAddr.String(), plancheck.ResourceActionReplace),
						plancheck.ExpectUnknownValue(bbdAddr.String(), tfjsonpath.New("version_latest")),
					},
				},
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(bbdAddr.String(), versionsPath,
						expectedVersions(resolvedWif(testconfig.OtherWifRunnerIssuer, replacementSubject, gcp), 1)),
				},
			},
		},
	})
}
