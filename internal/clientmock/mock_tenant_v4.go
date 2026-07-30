package clientmock

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"

	"github.com/meshcloud/terraform-provider-meshstack/client"
)

type MeshTenantV4Client struct {
	Store *Store[client.MeshTenantV4]
}

func (m MeshTenantV4Client) Read(_ context.Context, uuid string) (*client.MeshTenantV4, error) {
	t, ok := m.Store.Get(uuid)
	if !ok {
		return nil, nil
	}
	if t.Status.Lifecycle.State == client.TenantLifecycleStateMarkedForDeletion {
		deleted := *t
		deleted.Status.Lifecycle.State = client.TenantLifecycleStateDeleted
		deleted.Metadata.DeletedOn = new(time.Now().UTC().Format(time.RFC3339))
		m.Store.Set(uuid, &deleted)
	}
	return t, nil
}

func (m MeshTenantV4Client) ReadFunc(uuid string) func(ctx context.Context) (*client.MeshTenantV4, error) {
	return func(ctx context.Context) (*client.MeshTenantV4, error) {
		return m.Read(ctx, uuid)
	}
}

func (m MeshTenantV4Client) Create(_ context.Context, tenant *client.MeshTenantV4Create) (*client.MeshTenantV4, error) {
	id := uuid.NewString()

	// Simulate a successful tenant creation with platformTenantId set
	tenantName := tenant.Metadata.OwnedByWorkspace + "." + tenant.Metadata.OwnedByProject + "." + tenant.Spec.PlatformIdentifier

	// The mock applies requested quotas verbatim, so effective status.appliedQuotas mirrors spec.quotas.
	// Unlike the current tenant client this ignores landing-zone default quotas: the deprecated v4
	// resource identifies its landing zone by identifier, which the mock does not resolve to a stored one.
	var quotas []client.MeshTenantQuota
	if tenant.Spec.Quotas != nil {
		quotas = *tenant.Spec.Quotas
	}
	appliedQuotas := effectiveQuotas(nil, nil, quotas)

	created := &client.MeshTenantV4{
		Metadata: client.MeshTenantV4Metadata{
			Uuid:             id,
			OwnedByProject:   tenant.Metadata.OwnedByProject,
			OwnedByWorkspace: tenant.Metadata.OwnedByWorkspace,
			CreatedOn:        time.Now().UTC().Format(time.RFC3339),
		},
		Spec: client.MeshTenantV4Spec{
			PlatformIdentifier:    tenant.Spec.PlatformIdentifier,
			PlatformTenantId:      new(acctest.RandString(16)),
			LandingZoneIdentifier: tenant.Spec.LandingZoneIdentifier,
			Quotas:                tenant.Spec.Quotas,
		},
		Status: client.MeshTenantV4Status{
			TenantName:             tenantName,
			PlatformTypeIdentifier: "mock-platform-type",
			Tags:                   map[string][]string{},
			AppliedQuotas:          appliedQuotas,
			Lifecycle:              client.MeshTenantLifecycle{State: client.TenantLifecycleStateActive},
		},
	}

	m.Store.Set(id, created)
	return created, nil
}

func (m MeshTenantV4Client) Delete(_ context.Context, uuid string) error {
	t, ok := m.Store.Get(uuid)
	if !ok {
		return nil
	}
	marked := *t
	markedOn := time.Now().UTC().Format(time.RFC3339)
	marked.Metadata.MarkedForDeletionOn = &markedOn
	marked.Status.Lifecycle = client.MeshTenantLifecycle{
		State:             client.TenantLifecycleStateMarkedForDeletion,
		MarkedForDeletion: &client.MeshTenantLifecycleAction{Timestamp: markedOn},
	}
	m.Store.Set(uuid, &marked)
	return nil
}

func (m MeshTenantV4Client) List(_ context.Context, query client.MeshTenantV4Query) ([]client.MeshTenantV4, error) {
	var result []client.MeshTenantV4
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
		if query.Platform != nil && t.Spec.PlatformIdentifier != *query.Platform {
			continue
		}
		if query.PlatformType != nil && t.Status.PlatformTypeIdentifier != *query.PlatformType {
			continue
		}
		if query.LandingZone != nil && (t.Spec.LandingZoneIdentifier == nil || *t.Spec.LandingZoneIdentifier != *query.LandingZone) {
			continue
		}
		if query.PlatformTenant != nil && (t.Spec.PlatformTenantId == nil || *t.Spec.PlatformTenantId != *query.PlatformTenant) {
			continue
		}
		result = append(result, *t)
	}
	return result, nil
}
