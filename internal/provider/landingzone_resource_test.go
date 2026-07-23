package provider

import (
	"fmt"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/knownvalue"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
	"github.com/hashicorp/terraform-plugin-testing/statecheck"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	"github.com/hashicorp/terraform-plugin-testing/tfjsonpath"

	"github.com/meshcloud/terraform-provider-meshstack/client"
	"github.com/meshcloud/terraform-provider-meshstack/internal/provider/acctest/testconfig"
	"github.com/meshcloud/terraform-provider-meshstack/internal/provider/acctest/xknownvalue"
)

// TestAccLandingZoneBuildingBlockRefRequiresUuid asserts the plan-time validator rejects a
// building block ref object that is provided without a uuid (an assigned computed `.ref`, whose
// uuid is unknown at plan time, stays allowed).
func TestAccLandingZoneBuildingBlockRefRequiresUuid(t *testing.T) {
	config, _ := testconfig.LandingZoneAndWorkspace(t)

	badConfig := config.WithFirstBlock(
		testconfig.Descend("spec", "mandatory_building_block_refs")(
			testconfig.SetRawExpr(`[{ kind = "meshBuildingBlockDefinition" }]`)))

	ApplyAndTest(t, resource.TestCase{
		Steps: []resource.TestStep{
			{
				Config:      badConfig.String(),
				PlanOnly:    true,
				ExpectError: regexp.MustCompile(`(?s)uuid.*must be specified when`),
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

		tagConfig, tagAddr, tagKey := testconfig.TagDefinition(t, client.MeshObjectKind.LandingZone)
		restrictedTagConfig, _, _ := testconfig.RestrictedTagDefinitionWithDefault(t, client.MeshObjectKind.LandingZone, "injected-default")
		config, landingZoneAddr := testconfig.LandingZoneAndWorkspace(t)
		config = config.Join(tagConfig, restrictedTagConfig).WithFirstBlock(
			testconfig.Descend("metadata", "tags")(testconfig.SetRawExpr(`{ (%s) = ["blue"] }`, tagAddr.Join("spec", "key"))),
		)

		ApplyAndTest(t, resource.TestCase{
			Steps: []resource.TestStep{
				{
					Config: config.String(),
					ConfigStateChecks: []statecheck.StateCheck{
						statecheck.ExpectKnownValue(landingZoneAddr.String(), tfjsonpath.New("metadata").AtMapKey("tags"), knownvalue.MapExact(map[string]knownvalue.Check{
							tagKey: knownvalue.ListExact([]knownvalue.Check{knownvalue.StringExact("blue")}),
						})),
					},
					// Refresh reads back the injected superset; reconcileTrackedTags must reconcile it
					// away so no drift remains.
					ConfigPlanChecks: resource.ConfigPlanChecks{
						PostApplyPostRefresh: []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()},
					},
				},
			},
		})
	})

	t.Run("only_restricted_injected_no_declared_tags", func(t *testing.T) {
		// Original crash repro: no tags declared, backend injects a restricted default on create; it
		// must reconcile away to an empty map with no drift, rather than crash on inconsistent result.
		if IsMockClientTest() {
			t.Skip("relies on the backend injecting a restricted tag's default value on create")
		}

		restrictedTag, restrictedAddr, _ := testconfig.RestrictedTagDefinitionWithDefault(t, client.MeshObjectKind.LandingZone, "injected-default")
		config, landingZoneAddr := testconfig.LandingZoneAndWorkspace(t)
		// depends_on forces the tag definition to exist before the landing zone (so the restricted
		// default is actually injected on create) and to be torn down after it.
		config = config.Join(restrictedTag).WithFirstBlock(
			testconfig.Descend("depends_on")(testconfig.SetRawExpr("[%s]", restrictedAddr)),
		)

		ApplyAndTest(t, resource.TestCase{
			Steps: []resource.TestStep{
				{
					Config: config.String(),
					ConfigStateChecks: []statecheck.StateCheck{
						statecheck.ExpectKnownValue(landingZoneAddr.String(), tfjsonpath.New("metadata").AtMapKey("tags"), knownvalue.MapSizeExact(0)),
					},
					ConfigPlanChecks: resource.ConfigPlanChecks{
						PostApplyPostRefresh: []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()},
					},
				},
			},
		})
	})

	t.Run("declared_restricted_tag_kept", func(t *testing.T) {
		// A declared restricted tag is tracked and must round-trip; an undeclared injected restricted
		// default must still be reconciled away. Requires the acc identity to be allowed to set the
		// declared restricted tag's value.
		if IsMockClientTest() {
			t.Skip("relies on the backend injecting a restricted tag's default value on create")
		}

		nonRestrictedTag, nonRestrictedAddr, nonRestrictedKey := testconfig.TagDefinition(t, client.MeshObjectKind.LandingZone)
		declaredRestrictedTag, declaredRestrictedAddr, declaredRestrictedKey := testconfig.RestrictedTagDefinitionWithDefault(t, client.MeshObjectKind.LandingZone, "default-value")
		injectedRestrictedTag, injectedRestrictedAddr, _ := testconfig.RestrictedTagDefinitionWithDefault(t, client.MeshObjectKind.LandingZone, "injected-default")
		config, landingZoneAddr := testconfig.LandingZoneAndWorkspace(t)
		config = config.Join(nonRestrictedTag, declaredRestrictedTag, injectedRestrictedTag).WithFirstBlock(
			testconfig.Descend("metadata", "tags")(testconfig.SetRawExpr(
				`{ (%s) = ["blue"], (%s) = ["set-by-caller"] }`,
				nonRestrictedAddr.Join("spec", "key"),
				declaredRestrictedAddr.Join("spec", "key"),
			)),
			testconfig.Descend("depends_on")(testconfig.SetRawExpr("[%s, %s, %s]", nonRestrictedAddr, declaredRestrictedAddr, injectedRestrictedAddr)),
		)

		ApplyAndTest(t, resource.TestCase{
			Steps: []resource.TestStep{
				{
					Config: config.String(),
					ConfigStateChecks: []statecheck.StateCheck{
						// Both declared tags survive; only the undeclared injected restricted default is dropped.
						statecheck.ExpectKnownValue(landingZoneAddr.String(), tfjsonpath.New("metadata").AtMapKey("tags"), knownvalue.MapExact(map[string]knownvalue.Check{
							nonRestrictedKey:      knownvalue.ListExact([]knownvalue.Check{knownvalue.StringExact("blue")}),
							declaredRestrictedKey: knownvalue.ListExact([]knownvalue.Check{knownvalue.StringExact("set-by-caller")}),
						})),
					},
					ConfigPlanChecks: resource.ConfigPlanChecks{
						PostApplyPostRefresh: []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()},
					},
				},
			},
		})
	})

	config, landingZoneAddr := testconfig.LandingZoneAndWorkspace(t)
	resourceAddress := landingZoneAddr.String()

	updateConfig := config.WithFirstBlock(
		testconfig.Descend("spec", "display_name")(testconfig.SetString("Updated Landing Zone")))

	ApplyAndTest(t, resource.TestCase{
		Steps: []resource.TestStep{
			{
				Config: config.String(),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction(resourceAddress, plancheck.ResourceActionCreate),
					},
				},
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(resourceAddress, tfjsonpath.New("metadata").AtMapKey("owned_by_workspace"), xknownvalue.NotEmptyString()),
					statecheck.ExpectKnownValue(resourceAddress, tfjsonpath.New("spec").AtMapKey("display_name"), knownvalue.StringExact("My Custom Landing Zone")),
					statecheck.ExpectKnownValue(resourceAddress, tfjsonpath.New("ref").AtMapKey("kind"), knownvalue.StringExact("meshLandingZone")),
					statecheck.ExpectKnownValue(resourceAddress, tfjsonpath.New("ref").AtMapKey("name"), xknownvalue.NotEmptyString()),
				},
			},
			{
				Config: updateConfig.String(),
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
				ImportStateIdFunc: func(s *terraform.State) (string, error) {
					rs := s.RootModule().Resources[resourceAddress]
					if rs == nil {
						return "", fmt.Errorf("resource not found: %s", resourceAddress)
					}
					return rs.Primary.Attributes["metadata.name"], nil
				},
			},
		},
	})
}
