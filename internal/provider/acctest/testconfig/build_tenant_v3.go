package testconfig

import "testing"

// TenantV3 builds a tenant (v3) config using the provided prerequisite addresses.
func TenantV3(t *testing.T, projectAddr, platformAddr, landingZoneAddr Traversal) (config Config, tenantAddr Traversal) {
	t.Helper()
	return Resource{Name: "tenant"}.Config(t).WithFirstBlock(
		ExtractAddress(&tenantAddr),
		Descend("metadata")(
			Descend("owned_by_workspace")(SetAddr(projectAddr, "metadata", "owned_by_workspace")),
			Descend("owned_by_project")(SetAddr(projectAddr, "metadata", "name")),
			Descend("platform_identifier")(SetAddr(platformAddr, "identifier")),
		),
		Descend("spec")(
			Descend("landing_zone_identifier")(SetAddr(landingZoneAddr, "metadata", "name")),
		),
	), tenantAddr
}
