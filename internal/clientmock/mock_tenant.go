package clientmock

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/meshcloud/meshstack-cli/client"
)

type MeshTenantClient struct {
	Store *Store[client.MeshTenant]
	// LandingZoneStore lets Create resolve the assigned landing zone's default quotas, which the backend
	// merges into a tenant's effective quotas.
	LandingZoneStore *Store[client.MeshLandingZone]
}

func (m MeshTenantClient) Read(_ context.Context, uuid string) (*client.MeshTenant, error) {
	t, ok := m.Store.Get(uuid)
	if !ok {
		return nil, nil
	}
	if t.Status.Lifecycle.State == client.TenantLifecycleStateMarkedForDeletion {
		deleted := *t
		deleted.Status.Lifecycle.State = client.TenantLifecycleStateDeleted
		m.Store.Set(uuid, &deleted)
	}
	return t, nil
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
	appliedQuotas := effectiveQuotas(m.landingZoneDefaultQuotas(tenant.Spec.LandingZoneRef), tenant.Spec.RequestedQuotas)

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
		},
		Status: client.MeshTenantStatus{
			TenantName:             tenantName,
			PlatformTypeIdentifier: "mock-platform-type",
			PlatformWorkspaceId:    new("mock-platform-workspace-id"),
			Tags:                   map[string][]string{},
			AppliedQuotas:          appliedQuotas,
			Lifecycle:              client.MeshTenantLifecycle{State: client.TenantLifecycleStateActive},
		},
	}

	m.Store.Set(id, created)
	return created, nil
}

func (m MeshTenantClient) Delete(_ context.Context, uuid string) error {
	t, ok := m.Store.Get(uuid)
	if !ok {
		return nil
	}
	marked := *t
	marked.Status.Lifecycle = client.MeshTenantLifecycle{
		State:             client.TenantLifecycleStateMarkedForDeletion,
		MarkedForDeletion: &client.MeshTenantLifecycleAction{Timestamp: time.Now().UTC().Format(time.RFC3339)},
	}
	m.Store.Set(uuid, &marked)
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

// effectiveQuotas overlays the requested quotas on the assigned landing zone's defaults, as the backend
// does when it resolves status.appliedQuotas: a requested key wins over a default. Returns nil when
// neither side contributes a quota, so status renders as null rather than an empty map.
func effectiveQuotas(landingZoneDefaults map[string]int64, requested map[string]client.RequestQuotaValue) map[string]client.AppliedQuotaValue {
	out := make(map[string]client.AppliedQuotaValue, len(landingZoneDefaults)+len(requested))
	for k, v := range landingZoneDefaults {
		out[k] = client.AppliedQuotaValue{Value: v}
	}

	for k, v := range requested {
		out[k] = client.AppliedQuotaValue(v)
	}

	if len(out) == 0 {
		return nil
	}
	return out
}

func (m MeshTenantClient) List(_ context.Context, query client.MeshTenantQuery) ([]client.MeshTenant, error) {
	var result []client.MeshTenant
	for _, t := range m.Store.Values() {
		if t.Status.Lifecycle.State != client.TenantLifecycleStateActive {
			continue
		}
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
