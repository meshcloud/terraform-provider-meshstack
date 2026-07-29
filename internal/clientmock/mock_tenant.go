package clientmock

import (
	"context"

	"github.com/google/uuid"
	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"

	"github.com/meshcloud/terraform-provider-meshstack/client"
)

type MeshTenantClient struct {
	Store *Store[client.MeshTenant]
	// LandingZoneStore lets Create resolve the assigned landing zone's default quotas, which the backend
	// merges into a tenant's effective quotas.
	LandingZoneStore *Store[client.MeshLandingZone]
}

func (m MeshTenantClient) Read(_ context.Context, uuid string) (*client.MeshTenant, error) {
	if t, ok := m.Store.Get(uuid); ok {
		return t, nil
	}
	return nil, nil
}

func (m MeshTenantClient) ReadFunc(uuid string) func(ctx context.Context) (*client.MeshTenant, error) {
	return func(ctx context.Context) (*client.MeshTenant, error) {
		return m.Read(ctx, uuid)
	}
}

func (m MeshTenantClient) Create(_ context.Context, tenant *client.MeshTenantCreate) (*client.MeshTenant, error) {
	id := uuid.NewString()

	// Simulate a successful tenant creation with platformTenantId set
	tenantName := tenant.Metadata.OwnedByWorkspace + "." + tenant.Metadata.OwnedByProject + "." + tenant.Spec.PlatformRef.Uuid

	// The mock applies the requested quotas verbatim (it enforces no bounds or auto-approval threshold),
	// but does overlay them on the landing zone's default quotas as the backend does, so
	// status.appliedQuotas can legitimately carry keys the caller never requested.
	appliedQuotas := effectiveQuotas(m.landingZoneDefaultQuotas(tenant.Spec.LandingZoneRef), tenant.Spec.RequestedQuotas, tenant.Spec.Quotas) //nolint:staticcheck // the mock must keep serving the deprecated quotas form

	created := &client.MeshTenant{
		Metadata: client.MeshTenantMetadata{
			Uuid:             id,
			OwnedByProject:   tenant.Metadata.OwnedByProject,
			OwnedByWorkspace: tenant.Metadata.OwnedByWorkspace,
		},
		Spec: client.MeshTenantSpec{
			PlatformRef:      tenant.Spec.PlatformRef,
			PlatformTenantId: new(acctest.RandString(16)),
			LandingZoneRef:   tenant.Spec.LandingZoneRef,
			RequestedQuotas:  tenant.Spec.RequestedQuotas,
			Quotas:           tenant.Spec.Quotas, //nolint:staticcheck // the mock must keep serving the deprecated quotas form
		},
		Status: client.MeshTenantStatus{
			TenantName:             tenantName,
			PlatformTypeIdentifier: "mock-platform-type",
			PlatformWorkspaceId:    new("mock-platform-workspace-id"),
			Tags:                   map[string][]string{},
			AppliedQuotas:          appliedQuotas,
		},
	}

	m.Store.Set(id, created)
	return created, nil
}

func (m MeshTenantClient) Delete(_ context.Context, uuid string) error {
	m.Store.Delete(uuid)
	return nil
}

// landingZoneDefaultQuotas returns the default quotas of the landing zone a tenant is assigned to, or
// nil when the tenant has none, the store is not wired, or the landing zone is unknown to the mock.
func (m MeshTenantClient) landingZoneDefaultQuotas(ref *client.NamedRef) map[string]int64 {
	if ref == nil || m.LandingZoneStore == nil {
		return nil
	}
	landingZone, ok := m.LandingZoneStore.Get(ref.Name)
	if !ok || len(landingZone.Spec.Quotas) == 0 {
		return nil
	}
	defaults := make(map[string]int64, len(landingZone.Spec.Quotas))
	for _, q := range landingZone.Spec.Quotas {
		defaults[q.Key] = q.Value
	}
	return defaults
}

// effectiveQuotas resolves what a create request carried — the preferred key->value map or the
// deprecated {key, value} list — into the single map the backend reports back as status.appliedQuotas,
// overlaid on the assigned landing zone's defaults (a requested key wins over a default, mirroring the
// backend's resolveEffectiveQuotas). Returns nil when neither side contributes a quota, so status
// renders as null rather than an empty map.
func effectiveQuotas(landingZoneDefaults map[string]int64, requested map[string]client.RequestQuotaValue, quotas []client.MeshTenantQuota) map[string]client.AppliedQuotaValue {
	out := make(map[string]client.AppliedQuotaValue, len(landingZoneDefaults)+len(requested)+len(quotas))
	for k, v := range landingZoneDefaults {
		out[k] = client.AppliedQuotaValue{Value: v}
	}

	// The two requested forms are mutually exclusive; the map form wins when both are somehow set.
	switch {
	case requested != nil:
		for k, v := range requested {
			out[k] = client.AppliedQuotaValue(v)
		}
	default:
		for _, q := range quotas {
			out[q.Key] = client.AppliedQuotaValue{Value: q.Value}
		}
	}

	if len(out) == 0 {
		return nil
	}
	return out
}

func (m MeshTenantClient) List(_ context.Context, query client.MeshTenantQuery) ([]client.MeshTenant, error) {
	var result []client.MeshTenant
	for _, t := range m.Store.Values() {
		if t.Metadata.OwnedByWorkspace != query.Workspace {
			continue
		}
		if query.Project != nil && t.Metadata.OwnedByProject != *query.Project {
			continue
		}
		// The mock stores platform by ref (uuid); the backend resolves the platformIdentifier query
		// param to that uuid, so mock-mode callers filter by uuid here.
		if query.Platform != nil && t.Spec.PlatformRef.Uuid != *query.Platform {
			continue
		}
		if query.PlatformType != nil && t.Status.PlatformTypeIdentifier != *query.PlatformType {
			continue
		}
		if query.LandingZone != nil && (t.Spec.LandingZoneRef == nil || t.Spec.LandingZoneRef.Name != *query.LandingZone) {
			continue
		}
		if query.PlatformTenant != nil && (t.Spec.PlatformTenantId == nil || *t.Spec.PlatformTenantId != *query.PlatformTenant) {
			continue
		}
		result = append(result, *t)
	}
	return result, nil
}
