package clientmock

import (
	"context"

	"github.com/google/uuid"
	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"

	"github.com/meshcloud/terraform-provider-meshstack/client"
)

type MeshTenantClient struct {
	Store *Store[client.MeshTenant]
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
	tenantIdentifier := tenant.Metadata.OwnedByWorkspace + "." + tenant.Metadata.OwnedByProject + "." + tenant.Spec.PlatformRef.Uuid

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
			Quotas:           tenant.Spec.Quotas,
		},
		Status: client.MeshTenantStatus{
			TenantIdentifier:       tenantIdentifier,
			PlatformTypeIdentifier: "mock-platform-type",
			PlatformWorkspaceId:    new("mock-platform-workspace-id"),
			Tags:                   map[string][]string{},
		},
	}

	m.Store.Set(id, created)
	return created, nil
}

func (m MeshTenantClient) Delete(_ context.Context, uuid string) error {
	m.Store.Delete(uuid)
	return nil
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
