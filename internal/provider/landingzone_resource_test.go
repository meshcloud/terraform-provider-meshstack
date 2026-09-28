package provider

import (
	"fmt"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/knownvalue"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
	"github.com/hashicorp/terraform-plugin-testing/statecheck"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	"github.com/hashicorp/terraform-plugin-testing/tfjsonpath"
	"github.com/meshcloud/meshstack-cli/client"

	"github.com/meshcloud/terraform-provider-meshstack/examples"
	"github.com/meshcloud/terraform-provider-meshstack/internal/provider/acctest/xknownvalue"
)

// Address and tag-key prefixes of the blocks in
// examples/{resources,data-sources}/meshstack_landingzone/*-test-*.tf.
const (
	landingZoneResourceAddr      = "meshstack_landingzone.example"
	landingZoneDataSourceAddr    = "data.meshstack_landingzone.example"
	landingZoneTagKeyPrefix      = "test-key-lz-"
	landingZoneDeclaredTagKeyPfx = "test-key-lz-declared-"
	landingZoneNamePrefix        = "test-lz-"
)

// landingZoneStepConfig is a landing zone step with everything it stands on: the custom platform it
// targets, that platform's type, the mandatory building block definition it references, and the
// workspace owning all of them.
func landingZoneStepConfig(t *testing.T, index int, supportNames ...string) string {
	t.Helper()
	return examples.JoinTestStepConfigs(
		examples.Resource.TestStepConfig(t, "landingzone", index, append([]string{"bbd"}, supportNames...)...),
		examples.Resource.TestStepConfig(t, "platform", 8),
		examples.Resource.TestStepConfig(t, "platform_type", 1),
		examples.Resource.TestStepConfig(t, "workspace", 1, "variables", "prerequisites"),
	)
}

// TestAccLandingZoneBuildingBlockRefRequiresUuid asserts the plan-time validator rejects a
// building block ref object that is provided without a uuid (an assigned computed `.ref`, whose
// uuid is unknown at plan time, stays allowed — see TestAccBuildingBlock/04_tenant_moved_from_v1).
func TestAccLandingZoneBuildingBlockRefRequiresUuid(t *testing.T) {
	ApplyAndTest(t, resource.TestCase{
		Steps: []resource.TestStep{
			{
				Config:          landingZoneStepConfig(t, 3),
				ConfigVariables: SuffixVariables(acctest.RandString(8)),
				PlanOnly:        true,
				ExpectError:     regexp.MustCompile(`(?s)uuid.*must be specified when`),
			},
		},
	})
}

func TestAccLandingZone(t *testing.T) {
	t.Run("restricted_default_tag", func(t *testing.T) {
		// Backend-materialized default: the mock has no tag-restriction business logic, so it can't
		// reproduce TagService.applyLandingZoneTagsOnCreation injecting a restricted tag's default on
		// create. See the lock-step policy in the acceptance-testing skill.
		if IsMockClientTest() {
			t.Skip("relies on the backend injecting a restricted tag's default value on create")
		}

		suffix := acctest.RandString(8)

		ApplyAndTest(t, resource.TestCase{
			Steps: []resource.TestStep{
				{
					Config:          landingZoneStepConfig(t, 6, "tags", "restricted-tag"),
					ConfigVariables: SuffixVariables(suffix),
					ConfigStateChecks: []statecheck.StateCheck{
						statecheck.ExpectKnownValue(landingZoneResourceAddr, tfjsonpath.New("metadata").AtMapKey("tags"), knownvalue.MapExact(map[string]knownvalue.Check{
							landingZoneTagKeyPrefix + suffix: knownvalue.ListExact([]knownvalue.Check{knownvalue.StringExact("blue")}),
						})),
					},
					// Refresh reads back the injected superset; reconcileTrackedTags must reconcile it
					// away so no drift remains.
					ConfigPlanChecks: resource.ConfigPlanChecks{
						PostApplyPostRefresh: []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()},
					},
				},
			},
		}, TouchesExclusively(client.MeshObjectKind.LandingZone))
	})

	t.Run("only_restricted_injected_no_declared_tags", func(t *testing.T) {
		// Original crash repro: no tags declared, backend injects a restricted default on create; it
		// must reconcile away to an empty map with no drift, rather than crash on inconsistent result.
		if IsMockClientTest() {
			t.Skip("relies on the backend injecting a restricted tag's default value on create")
		}

		ApplyAndTest(t, resource.TestCase{
			Steps: []resource.TestStep{
				{
					Config:          landingZoneStepConfig(t, 7, "restricted-tag"),
					ConfigVariables: SuffixVariables(acctest.RandString(8)),
					ConfigStateChecks: []statecheck.StateCheck{
						statecheck.ExpectKnownValue(landingZoneResourceAddr, tfjsonpath.New("metadata").AtMapKey("tags"), knownvalue.MapSizeExact(0)),
					},
					ConfigPlanChecks: resource.ConfigPlanChecks{
						PostApplyPostRefresh: []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()},
					},
				},
			},
		}, TouchesExclusively(client.MeshObjectKind.LandingZone))
	})

	t.Run("declared_restricted_tag_kept", func(t *testing.T) {
		// A declared restricted tag is tracked and must round-trip; an undeclared injected restricted
		// default must still be reconciled away. Requires the acc identity to be allowed to set the
		// declared restricted tag's value.
		if IsMockClientTest() {
			t.Skip("relies on the backend injecting a restricted tag's default value on create")
		}

		suffix := acctest.RandString(8)

		ApplyAndTest(t, resource.TestCase{
			Steps: []resource.TestStep{
				{
					Config:          landingZoneStepConfig(t, 8, "tags", "restricted-tag", "declared-restricted-tag"),
					ConfigVariables: SuffixVariables(suffix),
					ConfigStateChecks: []statecheck.StateCheck{
						// Both declared tags survive; only the undeclared injected restricted default is dropped.
						statecheck.ExpectKnownValue(landingZoneResourceAddr, tfjsonpath.New("metadata").AtMapKey("tags"), knownvalue.MapExact(map[string]knownvalue.Check{
							landingZoneTagKeyPrefix + suffix:      knownvalue.ListExact([]knownvalue.Check{knownvalue.StringExact("blue")}),
							landingZoneDeclaredTagKeyPfx + suffix: knownvalue.ListExact([]knownvalue.Check{knownvalue.StringExact("set-by-caller")}),
						})),
					},
					ConfigPlanChecks: resource.ConfigPlanChecks{
						PostApplyPostRefresh: []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()},
					},
				},
			},
		}, TouchesExclusively(client.MeshObjectKind.LandingZone))
	})

	t.Run("restricted", func(t *testing.T) {
		vars := SuffixVariables(acctest.RandString(8))

		ApplyAndTest(t, resource.TestCase{
			Steps: []resource.TestStep{
				{
					// If no `spec.restricted` is specified, we expect the tf provider to use `false` by default
					// and the backend to return that same value in `status.restricted`. Note that the assertion on `status.restricted`
					// only has teeth if this test runs as an acceptance test against a real backend.
					Config:          landingZoneStepConfig(t, 1),
					ConfigVariables: vars,
					ConfigStateChecks: []statecheck.StateCheck{
						statecheck.ExpectKnownValue(landingZoneResourceAddr, tfjsonpath.New("spec").AtMapKey("restricted"), knownvalue.Bool(false)),
						statecheck.ExpectKnownValue(landingZoneResourceAddr, tfjsonpath.New("status").AtMapKey("restricted"), knownvalue.Bool(false)),
					},
				},
				{
					Config:          landingZoneStepConfig(t, 4),
					ConfigVariables: vars,
					ConfigPlanChecks: resource.ConfigPlanChecks{
						PreApply: []plancheck.PlanCheck{
							// Changing from restricted = false to restricted = true does an update-in-place (as opposed to replacing the resource):
							plancheck.ExpectResourceAction(landingZoneResourceAddr, plancheck.ResourceActionUpdate),
						},
					},
					ConfigStateChecks: []statecheck.StateCheck{
						statecheck.ExpectKnownValue(landingZoneResourceAddr, tfjsonpath.New("spec").AtMapKey("restricted"), knownvalue.Bool(true)),
						statecheck.ExpectKnownValue(landingZoneResourceAddr, tfjsonpath.New("status").AtMapKey("restricted"), knownvalue.Bool(true)),
					},
				},
				{
					Config:          landingZoneStepConfig(t, 5),
					ConfigVariables: vars,
					ConfigStateChecks: []statecheck.StateCheck{
						statecheck.ExpectKnownValue(landingZoneResourceAddr, tfjsonpath.New("spec").AtMapKey("restricted"), knownvalue.Bool(false)),
						statecheck.ExpectKnownValue(landingZoneResourceAddr, tfjsonpath.New("status").AtMapKey("restricted"), knownvalue.Bool(false)),
					},
				},
			},
		})
	})

	vars := SuffixVariables(acctest.RandString(8))

	ApplyAndTest(t, resource.TestCase{
		Steps: []resource.TestStep{
			{
				Config:          landingZoneStepConfig(t, 1),
				ConfigVariables: vars,
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction(landingZoneResourceAddr, plancheck.ResourceActionCreate),
					},
				},
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(landingZoneResourceAddr, tfjsonpath.New("metadata").AtMapKey("owned_by_workspace"), xknownvalue.NotEmptyString()),
					statecheck.ExpectKnownValue(landingZoneResourceAddr, tfjsonpath.New("spec").AtMapKey("display_name"), knownvalue.StringExact("My Custom Landing Zone")),
					statecheck.ExpectKnownValue(landingZoneResourceAddr, tfjsonpath.New("ref").AtMapKey("kind"), knownvalue.StringExact("meshLandingZone")),
					statecheck.ExpectKnownValue(landingZoneResourceAddr, tfjsonpath.New("ref").AtMapKey("name"), xknownvalue.NotEmptyString()),
				},
			},
			{
				Config:          landingZoneStepConfig(t, 2),
				ConfigVariables: vars,
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction(landingZoneResourceAddr, plancheck.ResourceActionUpdate),
					},
				},
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(landingZoneResourceAddr, tfjsonpath.New("spec").AtMapKey("display_name"), knownvalue.StringExact("Updated Landing Zone")),
				},
			},
			{
				ResourceName:    landingZoneResourceAddr,
				ImportState:     true,
				ImportStateKind: resource.ImportBlockWithID,
				ConfigVariables: vars,
				ImportStateIdFunc: func(s *terraform.State) (string, error) {
					rs := s.RootModule().Resources[landingZoneResourceAddr]
					if rs == nil {
						return "", fmt.Errorf("resource not found: %s", landingZoneResourceAddr)
					}
					return rs.Primary.Attributes["metadata.name"], nil
				},
			},
		},
	})
}
