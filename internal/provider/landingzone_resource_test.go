package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/knownvalue"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
	"github.com/hashicorp/terraform-plugin-testing/statecheck"
	"github.com/hashicorp/terraform-plugin-testing/tfjsonpath"
	"github.com/meshcloud/meshstack-cli/client"

	"github.com/meshcloud/terraform-provider-meshstack/internal/provider/acctest/xknownvalue"
)

// TestAccLandingZoneBuildingBlockRefRequiresUuid asserts the plan-time validator rejects a
// building block ref object that is provided without a uuid (an assigned computed `.ref`, whose
// uuid is unknown at plan time, stays allowed).
func TestAccLandingZoneBuildingBlockRefRequiresUuid(t *testing.T) {
	vars := platformTypeStepVars(acctest.RandString(8))

	ApplyAndTest(t, resource.TestCase{
		Steps: []resource.TestStep{
			{
				Config:      hclStep(t, "resources/landingzone/building_block_ref_requires_uuid", 1, vars),
				PlanOnly:    true,
				ExpectError: expectErrorWords("uuid", "specified"),
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
		vars := platformTypeStepVars(suffix)
		landingZoneAddr := "meshstack_landingzone.test"

		ApplyAndTest(t, resource.TestCase{
			Steps: []resource.TestStep{
				{
					Config: hclStep(t, "resources/landingzone/restricted_default_tag", 1, vars),
					ConfigStateChecks: []statecheck.StateCheck{
						statecheck.ExpectKnownValue(landingZoneAddr, tfjsonpath.New("metadata").AtMapKey("tags"), knownvalue.MapExact(map[string]knownvalue.Check{
							"test-key-" + suffix: knownvalue.ListExact([]knownvalue.Check{knownvalue.StringExact("blue")}),
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

		vars := platformTypeStepVars(acctest.RandString(8))
		landingZoneAddr := "meshstack_landingzone.test"

		ApplyAndTest(t, resource.TestCase{
			Steps: []resource.TestStep{
				{
					Config: hclStep(t, "resources/landingzone/only_restricted_injected_no_declared_tags", 1, vars),
				},
				{
					Config: hclStep(t, "resources/landingzone/only_restricted_injected_no_declared_tags", 2, vars),
					ConfigStateChecks: []statecheck.StateCheck{
						statecheck.ExpectKnownValue(landingZoneAddr, tfjsonpath.New("metadata").AtMapKey("tags"), knownvalue.MapSizeExact(0)),
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
		vars := platformTypeStepVars(suffix)
		landingZoneAddr := "meshstack_landingzone.test"

		ApplyAndTest(t, resource.TestCase{
			Steps: []resource.TestStep{
				{
					Config: hclStep(t, "resources/landingzone/declared_restricted_tag_kept", 1, vars),
				},
				{
					Config: hclStep(t, "resources/landingzone/declared_restricted_tag_kept", 2, vars),
					ConfigStateChecks: []statecheck.StateCheck{
						// Both declared tags survive; only the undeclared injected restricted default is dropped.
						statecheck.ExpectKnownValue(landingZoneAddr, tfjsonpath.New("metadata").AtMapKey("tags"), knownvalue.MapExact(map[string]knownvalue.Check{
							"test-key-" + suffix:          knownvalue.ListExact([]knownvalue.Check{knownvalue.StringExact("blue")}),
							"test-declared-key-" + suffix: knownvalue.ListExact([]knownvalue.Check{knownvalue.StringExact("set-by-caller")}),
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
		vars := platformTypeStepVars(acctest.RandString(8))
		addr := "meshstack_landingzone.test"

		ApplyAndTest(t, resource.TestCase{
			Steps: []resource.TestStep{
				{
					// If no `spec.restricted` is specified, we expect the tf provider to use `false` by default
					// and the backend to return that same value in `status.restricted`. Note that the assertion on `status.restricted`
					// only has teeth if this test runs as an acceptance test against a real backend.
					Config: hclStep(t, "resources/landingzone/restricted", 1, vars),
					ConfigStateChecks: []statecheck.StateCheck{
						statecheck.ExpectKnownValue(addr, tfjsonpath.New("spec").AtMapKey("restricted"), knownvalue.Bool(false)),
						statecheck.ExpectKnownValue(addr, tfjsonpath.New("status").AtMapKey("restricted"), knownvalue.Bool(false)),
					},
				},
				{
					Config: hclStep(t, "resources/landingzone/restricted", 2, vars),
					ConfigPlanChecks: resource.ConfigPlanChecks{
						PreApply: []plancheck.PlanCheck{
							// Changing from restricted = false to restricted = true does an update-in-place (as opposed to replacing the resource):
							plancheck.ExpectResourceAction(addr, plancheck.ResourceActionUpdate),
						},
					},
					ConfigStateChecks: []statecheck.StateCheck{
						statecheck.ExpectKnownValue(addr, tfjsonpath.New("spec").AtMapKey("restricted"), knownvalue.Bool(true)),
						statecheck.ExpectKnownValue(addr, tfjsonpath.New("status").AtMapKey("restricted"), knownvalue.Bool(true)),
					},
				},
				{
					Config: hclStep(t, "resources/landingzone/restricted", 3, vars),
					ConfigStateChecks: []statecheck.StateCheck{
						statecheck.ExpectKnownValue(addr, tfjsonpath.New("spec").AtMapKey("restricted"), knownvalue.Bool(false)),
						statecheck.ExpectKnownValue(addr, tfjsonpath.New("status").AtMapKey("restricted"), knownvalue.Bool(false)),
					},
				},
			},
		})
	})

	t.Run("01_aws", func(t *testing.T) {
		suffix := acctest.RandString(8)
		vars := hclStepVars{"suffix": suffix}
		resourceAddress := "meshstack_landingzone.example"
		landingZoneName := "test-lz-" + suffix

		ApplyAndTest(t, resource.TestCase{
			Steps: []resource.TestStep{
				{
					Config: hclStep(t, "resources/landingzone/01_aws", 1, vars),
				},
				{
					Config: hclStep(t, "resources/landingzone/01_aws", 2, vars),
					ConfigPlanChecks: resource.ConfigPlanChecks{
						PreApply: []plancheck.PlanCheck{
							plancheck.ExpectResourceAction(resourceAddress, plancheck.ResourceActionCreate),
						},
					},
					ConfigStateChecks: []statecheck.StateCheck{
						statecheck.ExpectKnownValue(resourceAddress, tfjsonpath.New("metadata").AtMapKey("name"), knownvalue.StringExact(landingZoneName)),
						statecheck.ExpectKnownValue(resourceAddress, tfjsonpath.New("metadata").AtMapKey("owned_by_workspace"), knownvalue.StringExact("test-ws-"+suffix)),
						statecheck.ExpectKnownValue(resourceAddress, tfjsonpath.New("metadata").AtMapKey("tags"), knownvalue.MapExact(map[string]knownvalue.Check{
							"confidentiality-" + suffix: knownvalue.ListExact([]knownvalue.Check{knownvalue.StringExact("internal")}),
							"environment-" + suffix:     knownvalue.ListExact([]knownvalue.Check{knownvalue.StringExact("dev"), knownvalue.StringExact("qa"), knownvalue.StringExact("test")}),
						})),
						statecheck.ExpectKnownValue(resourceAddress, tfjsonpath.New("spec").AtMapKey("display_name"), knownvalue.StringExact("My Landing Zone's Display Name")),
						statecheck.ExpectKnownValue(resourceAddress, tfjsonpath.New("spec").AtMapKey("platform_properties").AtMapKey("aws").AtMapKey("aws_target_org_unit_id"), knownvalue.StringExact("ou-lpzq-kmf17bec")),
						statecheck.ExpectKnownValue(resourceAddress, tfjsonpath.New("ref"), xknownvalue.MapExact(map[string]knownvalue.Check{
							"kind": knownvalue.StringExact("meshLandingZone"),
							"name": knownvalue.StringExact(landingZoneName),
						})),
					},
				},
				{
					ResourceName:    resourceAddress,
					ImportState:     true,
					ImportStateKind: resource.ImportBlockWithID,
					ImportStateId:   landingZoneName,
				},
			},
		})
	})

	t.Run("02_custom", func(t *testing.T) {
		suffix := acctest.RandString(8)
		vars := platformTypeStepVars(suffix)
		resourceAddress := "meshstack_landingzone.example"
		landingZoneName := "test-lz-" + suffix

		ApplyAndTest(t, resource.TestCase{
			Steps: []resource.TestStep{
				{
					Config: hclStep(t, "resources/landingzone/02_custom", 1, vars),
					ConfigPlanChecks: resource.ConfigPlanChecks{
						PreApply: []plancheck.PlanCheck{
							plancheck.ExpectResourceAction(resourceAddress, plancheck.ResourceActionCreate),
						},
					},
					ConfigStateChecks: []statecheck.StateCheck{
						statecheck.ExpectKnownValue(resourceAddress, tfjsonpath.New("metadata").AtMapKey("owned_by_workspace"), knownvalue.StringExact("test-ws-"+suffix)),
						statecheck.ExpectKnownValue(resourceAddress, tfjsonpath.New("spec").AtMapKey("display_name"), knownvalue.StringExact("My Custom Landing Zone")),
						statecheck.ExpectKnownValue(resourceAddress, tfjsonpath.New("ref").AtMapKey("kind"), knownvalue.StringExact("meshLandingZone")),
						statecheck.ExpectKnownValue(resourceAddress, tfjsonpath.New("ref").AtMapKey("name"), knownvalue.StringExact(landingZoneName)),
					},
				},
				{
					Config: hclStep(t, "resources/landingzone/02_custom", 2, vars),
					ConfigPlanChecks: resource.ConfigPlanChecks{
						PreApply: []plancheck.PlanCheck{
							plancheck.ExpectResourceAction(resourceAddress, plancheck.ResourceActionUpdate),
						},
					},
					ConfigStateChecks: []statecheck.StateCheck{
						statecheck.ExpectKnownValue(resourceAddress, tfjsonpath.New("spec").AtMapKey("display_name"), knownvalue.StringExact("Updated Landing Zone")),
					},
				},
				{
					ResourceName:    resourceAddress,
					ImportState:     true,
					ImportStateKind: resource.ImportBlockWithID,
					ImportStateId:   landingZoneName,
				},
			},
		})
	})
}
