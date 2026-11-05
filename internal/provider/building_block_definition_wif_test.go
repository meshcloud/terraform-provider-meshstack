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

// expectedVersions is the versions of a definition whose versions, numbered 1 to count, all run on one
// runner, so every entry carries the same identity.
func expectedVersions(wif knownvalue.Check, count int) knownvalue.Check {
	versions := make([]knownvalue.Check, count)
	for i := range versions {
		versions[i] = knownvalue.ObjectPartial(map[string]knownvalue.Check{
			"number":                       knownvalue.Int64Exact(int64(i + 1)),
			"workload_identity_federation": wif,
		})
	}
	return ascendingVersionsExact(versions...)
}

const otherWifRunnerIssuer = "https://oidc-other.example.com"

func TestAccBuildingBlockDefinitionWif(t *testing.T) {
	vars := hclStepVars{"suffix": acctest.RandString(8)}
	bbdAddr := "meshstack_building_block_definition.example_01_terraform"
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
				Config: hclStep(t, "resources/building_block_definition/wif", 1, vars),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction(bbdAddr, plancheck.ResourceActionCreate),
					},
				},
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(bbdAddr, latestWifPath,
						resolvedWif("https://oidc.example.com", resolvedSubject("system:serviceaccount:namespace:workspace."), gcp)),
					statecheck.ExpectKnownValue(bbdAddr, versionsPath,
						expectedVersions(resolvedWif("https://oidc.example.com", resolvedSubject("system:serviceaccount:namespace:workspace."), gcp), 1)),
					statecheck.ExpectKnownValue(bbdAddr, tfjsonpath.New("version_latest_release"), knownvalue.Null()),
				},
			},
			{
				// A changed runner changes the identity of the version it is changed on, so it is read again.
				// meshStack stores a version's secrets encrypted for its runner, so moving the definition to another
				// one has to re-supply every secret as plaintext - a hash is a 400. Bumping secret_version is how the
				// provider is asked for that, same as the rotation step of TestAccBuildingBlockDefinition.
				Config: hclStep(t, "resources/building_block_definition/wif", 2, vars),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction(bbdAddr, plancheck.ResourceActionUpdate),
						plancheck.ExpectUnknownValue(bbdAddr, latestWifPath),
					},
				},
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(bbdAddr, latestWifPath, otherRunnerWif),
					statecheck.ExpectKnownValue(bbdAddr, versionsPath, expectedVersions(otherRunnerWif, 1)),
				},
			},
			{
				// An unrelated change keeps it, instead of planning a pointless "known after apply".
				Config: hclStep(t, "resources/building_block_definition/wif", 3, vars),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction(bbdAddr, plancheck.ResourceActionUpdate),
						plancheck.ExpectKnownValue(bbdAddr, latestSubjectPath, otherRunnerSubject),
					},
				},
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(bbdAddr, latestWifPath, otherRunnerWif),
				},
			},
			{
				// A release changes the version in place, so the identity of the version stays as it was and
				// becomes the identity of the latest release.
				Config: hclStep(t, "resources/building_block_definition/wif", 4, vars),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction(bbdAddr, plancheck.ResourceActionUpdate),
						plancheck.ExpectKnownValue(bbdAddr, latestSubjectPath, otherRunnerSubject),
					},
				},
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(bbdAddr, latestWifPath, otherRunnerWif),
					statecheck.ExpectKnownValue(bbdAddr, latestReleaseWifPath, otherRunnerWif),
				},
			},
			{
				// A new draft cut from the released version adds an entry whose identity is read after it is
				// written, while the released version keeps its own.
				Config: hclStep(t, "resources/building_block_definition/wif", 5, vars),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction(bbdAddr, plancheck.ResourceActionUpdate),
						plancheck.ExpectUnknownValue(bbdAddr, latestWifPath),
						plancheck.ExpectKnownValue(bbdAddr, latestReleaseWifPath, otherRunnerWif),
					},
				},
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(bbdAddr, versionsPath, expectedVersions(otherRunnerWif, 2)),
					statecheck.ExpectKnownValue(bbdAddr, latestReleaseWifPath, otherRunnerWif),
					statecheck.ExpectKnownValue(bbdAddr, latestSubjectPath, recordSubject),
				},
			},
			{
				// A replacement creates a new definition with a single version and its own subject. The create
				// side of it plans every computed attribute unknown, the version entry included.
				Config: hclStep(t, "resources/building_block_definition/wif", 6, vars),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction(bbdAddr, plancheck.ResourceActionReplace),
						plancheck.ExpectUnknownValue(bbdAddr, tfjsonpath.New("version_latest")),
					},
				},
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(bbdAddr, versionsPath,
						expectedVersions(resolvedWif(otherWifRunnerIssuer, replacementSubject, gcp), 1)),
				},
			},
		},
	})
}
