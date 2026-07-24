package provider

import (
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/compare"
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

	// create_via_data_sources creates a tenant whose platform_ref and landing_zone_ref are resolved
	// from data sources rather than the resources directly, exercising the singular
	// (meshstack_platform / meshstack_landingzone) and plural (meshstack_platforms /
	// meshstack_landingzones) data sources equally — all four are still fully supported. The tenant is
	// fed from the plural platforms list (a one(...) select) and the singular landing zone; the
	// CompareValuePairs checks then assert the other-cardinality data source resolves to the same
	// object, so the plural element `ref` and the singular data-source `ref` are proven interchangeable
	// ({kind, uuid} for the platform, {kind, name} for the landing zone). The fresh workspace holds
	// exactly one platform and one landing zone, so the plural lists have a single element at index 0.
	t.Run("create_via_data_sources", func(t *testing.T) {
		workspaceConfig, workspaceAddr := testconfig.Workspace(t)
		projectConfig, projectAddr := testconfig.Project(t, workspaceAddr)
		platformConfig, platformAddr, platformTypeAddr := testconfig.CustomPlatform(t, workspaceAddr)
		landingZoneConfig, landingZoneAddr := testconfig.LandingZone(t, workspaceAddr, platformAddr, platformTypeAddr)

		var singularPlatformAddr, pluralPlatformsAddr, singularLandingZoneAddr, pluralLandingZonesAddr testconfig.Traversal

		// Singular data sources read by uuid / name — the resource references make them depend on the
		// platform / landing zone implicitly, so no explicit depends_on is needed.
		singularPlatform := testconfig.DataSource{Name: "platform"}.Config(t).WithFirstBlock(
			testconfig.ExtractAddress(&singularPlatformAddr),
			testconfig.Descend("metadata", "uuid")(testconfig.SetAddr(platformAddr, "metadata", "uuid")),
		)
		singularLandingZone := testconfig.DataSource{Name: "landingzone"}.Config(t).WithFirstBlock(
			testconfig.ExtractAddress(&singularLandingZoneAddr),
			testconfig.Descend("metadata", "name")(testconfig.SetAddr(landingZoneAddr, "metadata", "name")),
		)

		// Plural data sources filter by workspace / platform, which does not depend on the object rows
		// themselves, so they need an explicit depends_on to list after those rows exist.
		pluralPlatforms := testconfig.DataSource{Name: "platforms"}.Config(t).WithFirstBlock(
			testconfig.ExtractAddress(&pluralPlatformsAddr),
			testconfig.Descend("owned_by_workspace")(testconfig.SetAddr(workspaceAddr, "metadata", "name")),
			testconfig.Descend("depends_on")(testconfig.SetRawExpr("[%s]", platformAddr)),
		)
		pluralLandingZones := testconfig.DataSource{Name: "landingzones"}.Config(t).WithFirstBlock(
			testconfig.ExtractAddress(&pluralLandingZonesAddr),
			testconfig.Descend("platform_uuid")(testconfig.SetAddr(platformAddr, "metadata", "uuid")),
			testconfig.Descend("depends_on")(testconfig.SetRawExpr("[%s]", landingZoneAddr)),
		)

		tenantConfig, tenantAddr := testconfig.Tenant(t, projectAddr, platformAddr, landingZoneAddr)
		tenantConfig = tenantConfig.WithFirstBlock(
			testconfig.Descend("spec", "platform_ref")(testconfig.SetRawExpr(
				"one([for p in %s.platforms : p if p.metadata.uuid == %s]).ref",
				pluralPlatformsAddr, platformAddr.Join("metadata", "uuid"))),
			testconfig.Descend("spec", "landing_zone_ref")(testconfig.SetRawExpr(
				"%s.ref", singularLandingZoneAddr)),
		)

		config := tenantConfig.Join(
			singularPlatform, pluralPlatforms, singularLandingZone, pluralLandingZones,
			workspaceConfig, projectConfig, platformConfig, landingZoneConfig,
		)

		ApplyAndTest(t, resource.TestCase{
			Steps: []resource.TestStep{
				{
					Config: config.String(),
					ConfigStateChecks: []statecheck.StateCheck{
						statecheck.ExpectKnownValue(tenantAddr.String(), tfjsonpath.New("spec").AtMapKey("platform_ref").AtMapKey("uuid"), xknownvalue.NotEmptyString()),
						statecheck.ExpectKnownValue(tenantAddr.String(), tfjsonpath.New("spec").AtMapKey("platform_ref").AtMapKey("kind"), knownvalue.StringExact("meshPlatform")),
						statecheck.ExpectKnownValue(tenantAddr.String(), tfjsonpath.New("spec").AtMapKey("landing_zone_ref").AtMapKey("name"), xknownvalue.NotEmptyString()),
						statecheck.ExpectKnownValue(tenantAddr.String(), tfjsonpath.New("ref").AtMapKey("uuid"), xknownvalue.NotEmptyString()),

						// singular and plural data sources resolve to the same platform / landing zone.
						statecheck.CompareValuePairs(
							singularPlatformAddr.String(), tfjsonpath.New("ref").AtMapKey("uuid"),
							pluralPlatformsAddr.String(), tfjsonpath.New("platforms").AtSliceIndex(0).AtMapKey("ref").AtMapKey("uuid"),
							compare.ValuesSame(),
						),
						statecheck.CompareValuePairs(
							singularLandingZoneAddr.String(), tfjsonpath.New("ref").AtMapKey("name"),
							pluralLandingZonesAddr.String(), tfjsonpath.New("landing_zones").AtSliceIndex(0).AtMapKey("ref").AtMapKey("name"),
							compare.ValuesSame(),
						),
					},
				},
			},
		})
	})

	// quotas covers the create-only quota flow: a tenant requesting an in-bounds quota applies it, and
	// the effective quotas are read back from status.applied_quotas (distinct from the requested
	// spec.requested_quotas). Runs in both modes — the mock echoes the requested quota into status, the
	// real backend validates it against the platform quota definition and applies it.
	t.Run("quotas", func(t *testing.T) {
		config, tenantAddr := tenantQuotaConfig(t, 4000, 4000, 2000)

		quotaMap := knownvalue.MapExact(map[string]knownvalue.Check{
			"limits.cpu": knownvalue.ObjectExact(map[string]knownvalue.Check{
				"value": knownvalue.Int64Exact(2000),
			}),
		})

		ApplyAndTest(t, resource.TestCase{
			Steps: []resource.TestStep{
				{
					Config: config.String(),
					ConfigStateChecks: []statecheck.StateCheck{
						// Requested quotas echo the config verbatim (spec.requested_quotas is create-only, Optional).
						statecheck.ExpectKnownValue(tenantAddr.String(), tfjsonpath.New("spec").AtMapKey("requested_quotas"), quotaMap),
						// Effective quotas come from status.applied_quotas, populated by the backend.
						statecheck.ExpectKnownValue(tenantAddr.String(), tfjsonpath.New("status").AtMapKey("applied_quotas"), quotaMap),
					},
				},
			},
		})
	})

	// quotas_change_rejected asserts that changing spec.quotas on an existing tenant is rejected: the
	// meshTenant API is create/delete only, with no quota update endpoint, so the provider must surface a
	// clear plan-time error rather than silently no-op. This is a provider-side decision, so it runs in
	// both modes.
	t.Run("quotas_change_rejected", func(t *testing.T) {
		// Reuse the same base config (same prerequisite resources) and change only the tenant's requested
		// quota value, so step 2 is an in-place update of the existing tenant rather than a full replace.
		config, _ := tenantQuotaConfig(t, 4000, 4000, 2000)
		changedConfig := config.WithFirstBlock(
			testconfig.Descend("spec", "requested_quotas")(testconfig.SetRawExpr(`{ "limits.cpu" = { value = %d } }`, 3000)),
		)

		ApplyAndTest(t, resource.TestCase{
			Steps: []resource.TestStep{
				{
					Config: config.String(),
				},
				{
					Config:      changedConfig.String(),
					ExpectError: regexp.MustCompile("Tenants can't be updated"),
				},
			},
		})
	})

	// quotas_out_of_range asserts the backend's create-time guardrail surfaces as a clear error: a
	// requested quota above the platform's max is rejected with HTTP 400, and the provider bubbles up the
	// descriptive API message. The mock does not enforce bounds, so this is acceptance-only.
	t.Run("quotas_out_of_range", func(t *testing.T) {
		if IsMockClientTest() {
			t.Skip("quota bounds are enforced by the backend; requires a real meshStack")
		}
		config, _ := tenantQuotaConfig(t, 100, 100, 101)

		ApplyAndTest(t, resource.TestCase{
			Steps: []resource.TestStep{
				{
					Config:      config.String(),
					ExpectError: regexp.MustCompile(`is out of range`),
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

// tenantQuotaConfig builds a full tenant config whose platform defines a `limits.cpu` quota with the
// given max/threshold (min is fixed at 1) and whose tenant requests requestedCpu for that quota. The
// bespoke quota_definitions and spec.requested_quotas are layered onto the standard builders via SetRawExpr.
func tenantQuotaConfig(t *testing.T, maxCpu, threshold, requestedCpu int64) (testconfig.Config, testconfig.Traversal) {
	t.Helper()
	workspaceConfig, workspaceAddr := testconfig.Workspace(t)
	projectConfig, projectAddr := testconfig.Project(t, workspaceAddr)
	platformConfig, platformAddr, platformTypeAddr := testconfig.CustomPlatform(t, workspaceAddr)
	platformConfig = platformConfig.WithFirstBlock(
		testconfig.Descend("spec", "quota_definitions")(testconfig.SetRawExpr(
			`[{ quota_key = "limits.cpu", min_value = 1, max_value = %d, unit = "cores", auto_approval_threshold = %d, description = "vCPU limit", label = "CPU" }]`,
			maxCpu, threshold,
		)),
	)
	landingZoneConfig, landingZoneAddr := testconfig.LandingZone(t, workspaceAddr, platformAddr, platformTypeAddr)
	tenantConfig, tenantAddr := testconfig.Tenant(t, projectAddr, platformAddr, landingZoneAddr)
	tenantConfig = tenantConfig.WithFirstBlock(
		testconfig.Descend("spec", "requested_quotas")(testconfig.SetRawExpr(
			`{ "limits.cpu" = { value = %d } }`, requestedCpu,
		)),
	)
	config := tenantConfig.Join(workspaceConfig, projectConfig, platformConfig, landingZoneConfig)
	return config, tenantAddr
}
