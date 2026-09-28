package provider

import (
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/compare"
	tfconfig "github.com/hashicorp/terraform-plugin-testing/config"
	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/knownvalue"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
	"github.com/hashicorp/terraform-plugin-testing/statecheck"
	"github.com/hashicorp/terraform-plugin-testing/tfjsonpath"

	"github.com/meshcloud/terraform-provider-meshstack/examples"
	"github.com/meshcloud/terraform-provider-meshstack/internal/provider/acctest/xknownvalue"
)

// Addresses of the blocks in examples/{resources,data-sources}/meshstack_tenant/*-test-*.tf and the
// data sources resource-test-2.tf declares alongside the tenant.
const (
	tenantResourceAddr           = "meshstack_tenant.example"
	tenantDataSourceAddr         = "data.meshstack_tenant.example"
	tenantPlatformDataSourceAddr = "data.meshstack_platform.example"
	tenantLandingZoneDsAddr      = "data.meshstack_landingzone.example"
)

// tenantStepConfig is a tenant step with everything under it: the project it belongs to, the
// platform and landing zone it targets, the platform type and building block definition those need,
// and the workspace owning all of them.
//
// platformIndex picks the platform variant — 8 is the plain custom one, 11 the one carrying quota
// definitions; landingZoneIndex likewise picks 1 or the quota-bearing 9.
func tenantStepConfig(t *testing.T, index, platformIndex, landingZoneIndex int) string {
	t.Helper()
	return examples.JoinTestStepConfigs(
		examples.Resource.TestStepConfig(t, "tenant", index, "quota-variables"),
		examples.Resource.TestStepConfig(t, "project", 1),
		examples.Resource.TestStepConfig(t, "landingzone", landingZoneIndex, "bbd"),
		examples.Resource.TestStepConfig(t, "platform", platformIndex),
		examples.Resource.TestStepConfig(t, "platform_type", 1),
		examples.Resource.TestSupportConfigs(t, "project", "prerequisites"),
	)
}

// tenantQuotaVariables are the quota bounds a tenant quota case runs with, on top of the run suffix.
// A zero lzMemoryDefault means no limits.memory quota is defined at all.
type tenantQuotaVariables struct {
	maxCpu                int64
	autoApprovalThreshold int64
	requestedCpu          int64
	lzMemoryDefault       int64
}

func (q tenantQuotaVariables) variables(suffix string) tfconfig.Variables {
	vars := SuffixVariables(suffix)
	vars["max_cpu"] = tfconfig.IntegerVariable(q.maxCpu)
	vars["cpu_auto_approval_threshold"] = tfconfig.IntegerVariable(q.autoApprovalThreshold)
	vars["requested_cpu"] = tfconfig.IntegerVariable(q.requestedCpu)
	vars["lz_memory_default"] = tfconfig.IntegerVariable(q.lzMemoryDefault)
	return vars
}

func TestAccTenant(t *testing.T) {
	t.Parallel()

	// create covers the plain create path of the unsuffixed meshstack_tenant on the v4 body.
	t.Run("create", func(t *testing.T) {
		ApplyAndTest(t, resource.TestCase{
			Steps: []resource.TestStep{
				{
					Config:          tenantStepConfig(t, 1, 8, 1),
					ConfigVariables: SuffixVariables(acctest.RandString(8)),
					ConfigPlanChecks: resource.ConfigPlanChecks{
						PreApply: []plancheck.PlanCheck{
							plancheck.ExpectResourceAction(tenantResourceAddr, plancheck.ResourceActionCreate),
						},
					},
					ConfigStateChecks: []statecheck.StateCheck{
						// Ref
						statecheck.ExpectKnownValue(tenantResourceAddr, tfjsonpath.New("ref").AtMapKey("kind"), knownvalue.StringExact("meshTenant")),
						statecheck.ExpectKnownValue(tenantResourceAddr, tfjsonpath.New("ref").AtMapKey("uuid"), xknownvalue.NotEmptyString()),

						// Metadata
						statecheck.ExpectKnownValue(tenantResourceAddr, tfjsonpath.New("metadata").AtMapKey("uuid"), xknownvalue.NotEmptyString()),
						statecheck.ExpectKnownValue(tenantResourceAddr, tfjsonpath.New("metadata").AtMapKey("owned_by_workspace"), xknownvalue.NotEmptyString()),
						statecheck.ExpectKnownValue(tenantResourceAddr, tfjsonpath.New("metadata").AtMapKey("owned_by_project"), xknownvalue.NotEmptyString()),

						// Spec
						statecheck.ExpectKnownValue(tenantResourceAddr, tfjsonpath.New("spec").AtMapKey("platform_ref").AtMapKey("uuid"), xknownvalue.NotEmptyString()),
						statecheck.ExpectKnownValue(tenantResourceAddr, tfjsonpath.New("spec").AtMapKey("platform_ref").AtMapKey("kind"), knownvalue.StringExact("meshPlatform")),
						statecheck.ExpectKnownValue(tenantResourceAddr, tfjsonpath.New("spec").AtMapKey("landing_zone_ref").AtMapKey("name"), xknownvalue.NotEmptyString()),

						// Status
						statecheck.ExpectKnownValue(tenantResourceAddr, tfjsonpath.New("status").AtMapKey("tenant_name"), xknownvalue.NotEmptyString()),
						statecheck.ExpectKnownValue(tenantResourceAddr, tfjsonpath.New("status").AtMapKey("platform_type_identifier"), xknownvalue.NotEmptyString()),
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
		ApplyAndTest(t, resource.TestCase{
			Steps: []resource.TestStep{
				{
					Config:          tenantStepConfig(t, 2, 8, 1),
					ConfigVariables: SuffixVariables(acctest.RandString(8)),
					ConfigStateChecks: []statecheck.StateCheck{
						statecheck.ExpectKnownValue(tenantResourceAddr, tfjsonpath.New("spec").AtMapKey("platform_ref").AtMapKey("uuid"), xknownvalue.NotEmptyString()),
						statecheck.ExpectKnownValue(tenantResourceAddr, tfjsonpath.New("spec").AtMapKey("platform_ref").AtMapKey("kind"), knownvalue.StringExact("meshPlatform")),
						statecheck.ExpectKnownValue(tenantResourceAddr, tfjsonpath.New("spec").AtMapKey("landing_zone_ref").AtMapKey("name"), xknownvalue.NotEmptyString()),
						statecheck.ExpectKnownValue(tenantResourceAddr, tfjsonpath.New("ref").AtMapKey("uuid"), xknownvalue.NotEmptyString()),

						// singular and plural data sources resolve to the same platform / landing zone.
						statecheck.CompareValuePairs(
							tenantPlatformDataSourceAddr, tfjsonpath.New("ref").AtMapKey("uuid"),
							platformsDataSourceAddr, tfjsonpath.New("platforms").AtSliceIndex(0).AtMapKey("ref").AtMapKey("uuid"),
							compare.ValuesSame(),
						),
						statecheck.CompareValuePairs(
							tenantLandingZoneDsAddr, tfjsonpath.New("ref").AtMapKey("name"),
							landingZonesDataSourceAddr, tfjsonpath.New("landing_zones").AtSliceIndex(0).AtMapKey("ref").AtMapKey("name"),
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
		quotaMap := knownvalue.MapExact(map[string]knownvalue.Check{
			"limits.cpu": knownvalue.ObjectExact(map[string]knownvalue.Check{
				"value": knownvalue.Int64Exact(2000),
			}),
		})

		ApplyAndTest(t, resource.TestCase{
			Steps: []resource.TestStep{
				{
					Config: tenantStepConfig(t, 3, 11, 1),
					ConfigVariables: tenantQuotaVariables{
						maxCpu: 4000, autoApprovalThreshold: 4000, requestedCpu: 2000,
					}.variables(acctest.RandString(8)),
					ConfigStateChecks: []statecheck.StateCheck{
						// Requested quotas echo the config verbatim (spec.requested_quotas is create-only, Optional).
						statecheck.ExpectKnownValue(tenantResourceAddr, tfjsonpath.New("spec").AtMapKey("requested_quotas"), quotaMap),
						// Effective quotas come from status.applied_quotas, populated by the backend.
						statecheck.ExpectKnownValue(tenantResourceAddr, tfjsonpath.New("status").AtMapKey("applied_quotas"), quotaMap),
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
		// Same config and the same suffix, so step 2 is an in-place update of the existing tenant rather
		// than a full replace; only the requested quota value differs.
		suffix := acctest.RandString(8)
		config := tenantStepConfig(t, 3, 11, 1)

		ApplyAndTest(t, resource.TestCase{
			Steps: []resource.TestStep{
				{
					Config: config,
					ConfigVariables: tenantQuotaVariables{
						maxCpu: 4000, autoApprovalThreshold: 4000, requestedCpu: 2000,
					}.variables(suffix),
				},
				{
					Config: config,
					ConfigVariables: tenantQuotaVariables{
						maxCpu: 4000, autoApprovalThreshold: 4000, requestedCpu: 3000,
					}.variables(suffix),
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

		ApplyAndTest(t, resource.TestCase{
			Steps: []resource.TestStep{
				{
					Config: tenantStepConfig(t, 3, 11, 1),
					ConfigVariables: tenantQuotaVariables{
						maxCpu: 100, autoApprovalThreshold: 100, requestedCpu: 101,
					}.variables(acctest.RandString(8)),
					ExpectError: regexp.MustCompile(`is out of range`),
				},
			},
		})
	})

	// quotas_above_auto_approval_threshold_admin_key covers the second create-time guardrail from the side
	// this suite can actually exercise. A request beyond the platform's auto-approval threshold is refused
	// outright rather than queued for operator approval — but only for non-admin callers; a key holding
	// ADM_TENANT_SAVE may exceed the threshold, since it also defines the platform's quota limits. Every
	// key that can run this suite is such an admin key (creating platforms and workspaces requires it), so
	// the assertion here is that the above-threshold request goes through and is applied as configured. A
	// backend change that started rejecting admin callers would break provider users the same way and must
	// fail here. The non-admin rejection is regression-tested backend-side, in meshfed-release's
	// MeshTenantQuotaBoundsScenarios, which mints a key scoped to PUBLIC_API + TENANT_SAVE.
	t.Run("quotas_above_auto_approval_threshold_admin_key", func(t *testing.T) {
		if IsMockClientTest() {
			t.Skip("the auto-approval threshold is enforced by the backend; requires a real meshStack")
		}

		ApplyAndTest(t, resource.TestCase{
			Steps: []resource.TestStep{
				{
					Config: tenantStepConfig(t, 3, 11, 1),
					ConfigVariables: tenantQuotaVariables{
						maxCpu: 8000, autoApprovalThreshold: 2000, requestedCpu: 4000,
					}.variables(acctest.RandString(8)),
					ConfigStateChecks: []statecheck.StateCheck{
						statecheck.ExpectKnownValue(tenantResourceAddr, tfjsonpath.New("status").AtMapKey("applied_quotas"),
							knownvalue.MapExact(map[string]knownvalue.Check{
								"limits.cpu": knownvalue.ObjectExact(map[string]knownvalue.Check{
									"value": knownvalue.Int64Exact(4000),
								}),
							})),
					},
				},
			},
		})
	})

	// quotas_landing_zone_defaults covers the case where the effective quotas legitimately differ from
	// what was requested: meshStack merges the landing zone's default quotas into the tenant's quotas, so
	// status.applied_quotas is a strict superset of spec.requested_quotas — it also carries limits.memory,
	// which the tenant never requested. The second step asserts this does not turn into perpetual drift:
	// spec keeps the configured request (it is Optional, echoed from state) while status keeps the backend
	// truth, and the plan is empty. Runs in both modes — the mock overlays landing-zone defaults as the
	// backend does.
	t.Run("quotas_landing_zone_defaults", func(t *testing.T) {
		// Landing zone variant 9 carries the default quota; platform variant 11 defines both bounds.
		config := tenantStepConfig(t, 3, 11, 9)
		vars := tenantQuotaVariables{
			maxCpu: 4000, autoApprovalThreshold: 4000, requestedCpu: 2000, lzMemoryDefault: 8192,
		}.variables(acctest.RandString(8))

		requestedQuotas := knownvalue.MapExact(map[string]knownvalue.Check{
			"limits.cpu": knownvalue.ObjectExact(map[string]knownvalue.Check{
				"value": knownvalue.Int64Exact(2000),
			}),
		})
		appliedQuotas := knownvalue.MapExact(map[string]knownvalue.Check{
			"limits.cpu": knownvalue.ObjectExact(map[string]knownvalue.Check{
				"value": knownvalue.Int64Exact(2000),
			}),
			"limits.memory": knownvalue.ObjectExact(map[string]knownvalue.Check{
				"value": knownvalue.Int64Exact(8192),
			}),
		})

		quotaStateChecks := []statecheck.StateCheck{
			statecheck.ExpectKnownValue(tenantResourceAddr, tfjsonpath.New("spec").AtMapKey("requested_quotas"), requestedQuotas),
			statecheck.ExpectKnownValue(tenantResourceAddr, tfjsonpath.New("status").AtMapKey("applied_quotas"), appliedQuotas),
		}

		ApplyAndTest(t, resource.TestCase{
			Steps: []resource.TestStep{
				{
					Config:            config,
					ConfigVariables:   vars,
					ConfigStateChecks: quotaStateChecks,
				},
				{
					// Refresh reads the landing-zone-defaulted applied quotas back; the plan must stay empty.
					Config:            config,
					ConfigVariables:   vars,
					PlanOnly:          true,
					ConfigStateChecks: quotaStateChecks,
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

		vars := SuffixVariables(acctest.RandString(8))

		ApplyAndTest(t, resource.TestCase{
			Steps: []resource.TestStep{
				{
					Config:          tenantStepConfig(t, 1, 8, 1),
					ConfigVariables: vars,
				},
				{
					// Step 4 differs from step 1 only in platform_ref's uuid.
					Config:          tenantStepConfig(t, 4, 8, 1),
					ConfigVariables: vars,
					ConfigPlanChecks: resource.ConfigPlanChecks{
						PreApply: []plancheck.PlanCheck{
							plancheck.ExpectResourceAction(tenantResourceAddr, plancheck.ResourceActionReplace),
						},
					},
				},
			},
		})
	})
}
