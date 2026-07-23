package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/knownvalue"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
	"github.com/hashicorp/terraform-plugin-testing/statecheck"
	"github.com/hashicorp/terraform-plugin-testing/tfjsonpath"

	"github.com/meshcloud/terraform-provider-meshstack/internal/provider/acctest/testconfig"
	"github.com/meshcloud/terraform-provider-meshstack/internal/provider/acctest/xknownvalue"
)

func TestAccTenant(t *testing.T) {
	t.Parallel()

	// create covers the plain create path of the unsuffixed meshstack_tenant on the v4 body.
	t.Run("create", func(t *testing.T) {
		workspaceConfig, workspaceAddr := testconfig.Workspace(t)
		projectConfig, projectAddr := testconfig.Project(t, workspaceAddr)
		platformConfig, platformAddr, platformTypeAddr := testconfig.CustomPlatform(t, workspaceAddr)
		landingZoneConfig, landingZoneAddr := testconfig.LandingZone(t, workspaceAddr, platformAddr, platformTypeAddr)
		tenantConfig, tenantAddr := testconfig.Tenant(t, projectAddr, platformAddr, landingZoneAddr)

		config := tenantConfig.Join(workspaceConfig, projectConfig, platformConfig, landingZoneConfig)

		ApplyAndTest(t, resource.TestCase{
			Steps: []resource.TestStep{
				{
					Config: config.String(),
					ConfigPlanChecks: resource.ConfigPlanChecks{
						PreApply: []plancheck.PlanCheck{
							plancheck.ExpectResourceAction(tenantAddr.String(), plancheck.ResourceActionCreate),
						},
					},
					ConfigStateChecks: []statecheck.StateCheck{
						// Ref
						statecheck.ExpectKnownValue(tenantAddr.String(), tfjsonpath.New("ref").AtMapKey("kind"), knownvalue.StringExact("meshTenant")),
						statecheck.ExpectKnownValue(tenantAddr.String(), tfjsonpath.New("ref").AtMapKey("uuid"), xknownvalue.NotEmptyString()),

						// Metadata
						statecheck.ExpectKnownValue(tenantAddr.String(), tfjsonpath.New("metadata").AtMapKey("uuid"), xknownvalue.NotEmptyString()),
						statecheck.ExpectKnownValue(tenantAddr.String(), tfjsonpath.New("metadata").AtMapKey("owned_by_workspace"), xknownvalue.NotEmptyString()),
						statecheck.ExpectKnownValue(tenantAddr.String(), tfjsonpath.New("metadata").AtMapKey("owned_by_project"), xknownvalue.NotEmptyString()),

						// Spec
						statecheck.ExpectKnownValue(tenantAddr.String(), tfjsonpath.New("spec").AtMapKey("platform_ref").AtMapKey("uuid"), xknownvalue.NotEmptyString()),
						statecheck.ExpectKnownValue(tenantAddr.String(), tfjsonpath.New("spec").AtMapKey("platform_ref").AtMapKey("kind"), knownvalue.StringExact("meshPlatform")),
						statecheck.ExpectKnownValue(tenantAddr.String(), tfjsonpath.New("spec").AtMapKey("landing_zone_ref").AtMapKey("name"), xknownvalue.NotEmptyString()),

						// Status
						statecheck.ExpectKnownValue(tenantAddr.String(), tfjsonpath.New("status").AtMapKey("tenant_name"), xknownvalue.NotEmptyString()),
						statecheck.ExpectKnownValue(tenantAddr.String(), tfjsonpath.New("status").AtMapKey("platform_type_identifier"), xknownvalue.NotEmptyString()),
					},
				},
			},
		})
	})

	// requires_replace asserts that changing platform_ref forces a replacement rather than an
	// in-place update. platform_ref (and landing_zone_ref) carry the RequiresReplace plan modifier
	// applied centrally by the meshRef helper (schema_utils.go) — a tenant cannot move platforms in
	// place. This is a provider-side plan decision, so it runs in mock mode; the synthetic platform
	// uuid never has to exist because the plan action is decided before any backend validation.
	t.Run("requires_replace", func(t *testing.T) {
		if !IsMockClientTest() {
			t.Skip("asserts a provider-side plan decision (RequiresReplace); mock-only")
		}
		workspaceConfig, workspaceAddr := testconfig.Workspace(t)
		projectConfig, projectAddr := testconfig.Project(t, workspaceAddr)
		platformConfig, platformAddr, platformTypeAddr := testconfig.CustomPlatform(t, workspaceAddr)
		landingZoneConfig, landingZoneAddr := testconfig.LandingZone(t, workspaceAddr, platformAddr, platformTypeAddr)
		tenantConfig, tenantAddr := testconfig.Tenant(t, projectAddr, platformAddr, landingZoneAddr)

		config := tenantConfig.Join(workspaceConfig, projectConfig, platformConfig, landingZoneConfig)

		// Point platform_ref at a different uuid; only its value changes between steps.
		replacedConfig := config.WithFirstBlock(
			testconfig.Descend("spec", "platform_ref")(testconfig.SetRawExpr(
				`{ kind = "meshPlatform", uuid = "11111111-1111-1111-1111-111111111111" }`,
			)),
		)

		ApplyAndTest(t, resource.TestCase{
			Steps: []resource.TestStep{
				{
					Config: config.String(),
				},
				{
					Config: replacedConfig.String(),
					ConfigPlanChecks: resource.ConfigPlanChecks{
						PreApply: []plancheck.PlanCheck{
							plancheck.ExpectResourceAction(tenantAddr.String(), plancheck.ResourceActionReplace),
						},
					},
				},
			},
		})
	})

	// moved_from_v4 asserts a `moved` block migrates the deprecated (identifier-based)
	// meshstack_tenant_v4 to the ref-based meshstack_tenant without recreating the tenant. The state
	// mover carries over the tenant uuid; the post-move refresh Read re-reads the tenant by uuid and
	// fills in the ref-based platform_ref/landing_zone_ref/status (the framework gives the mover no API
	// client). On the real backend both resources address the same meshTenant object, so the migrated
	// state matches the target config and the move plans as a no-op. This cross-resource re-read cannot
	// run in mock mode: the v4 (identifier) and unsuffixed (ref) mock clients back separate stores, so
	// the tenant created via meshstack_tenant_v4 is not visible to the meshstack_tenant client.
	t.Run("moved_from_v4", func(t *testing.T) {
		if IsMockClientTest() {
			t.Skip("state-mover re-reads the tenant across the two (separate-store) mock clients; requires a real meshStack")
		}
		workspaceConfig, workspaceAddr := testconfig.Workspace(t)
		projectConfig, projectAddr := testconfig.Project(t, workspaceAddr)
		platformConfig, platformAddr, platformTypeAddr := testconfig.CustomPlatform(t, workspaceAddr)
		landingZoneConfig, landingZoneAddr := testconfig.LandingZone(t, workspaceAddr, platformAddr, platformTypeAddr)
		deps := workspaceConfig.Join(projectConfig, platformConfig, landingZoneConfig)

		v4Config, v4Addr := testconfig.TenantV4(t, projectAddr, platformAddr, landingZoneAddr)

		movedConfig, tenantAddr := testconfig.Tenant(t, projectAddr, platformAddr, landingZoneAddr)
		movedConfig = movedConfig.WithRawBlock("moved {\n  from = " + v4Addr.String() + "\n  to = " + tenantAddr.String() + "\n}")

		ApplyAndTest(t, resource.TestCase{
			Steps: []resource.TestStep{
				{
					Config: v4Config.Join(deps).String(),
				},
				{
					Config: movedConfig.Join(deps).String(),
					ConfigPlanChecks: resource.ConfigPlanChecks{
						PreApply: []plancheck.PlanCheck{
							plancheck.ExpectResourceAction(tenantAddr.String(), plancheck.ResourceActionNoop),
						},
					},
					ConfigStateChecks: []statecheck.StateCheck{
						statecheck.ExpectKnownValue(tenantAddr.String(), tfjsonpath.New("ref").AtMapKey("kind"), knownvalue.StringExact("meshTenant")),
						statecheck.ExpectKnownValue(tenantAddr.String(), tfjsonpath.New("spec").AtMapKey("platform_ref").AtMapKey("uuid"), xknownvalue.NotEmptyString()),
					},
				},
			},
		})
	})
}
