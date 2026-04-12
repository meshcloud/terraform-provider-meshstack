package clientmock

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"

	"github.com/meshcloud/terraform-provider-meshstack/client"
)

type MeshTenantV4Client struct {
	Store Store[client.MeshTenantV4]
}

func (m MeshTenantV4Client) Read(_ context.Context, uuid string) (*client.MeshTenantV4, error) {
	if t, ok := m.Store[uuid]; ok {
		return t, nil
	}
	return nil, nil
}

func (m MeshTenantV4Client) ReadFunc(uuid string) func(ctx context.Context) (*client.MeshTenantV4, error) {
	return func(ctx context.Context) (*client.MeshTenantV4, error) {
		return m.Read(ctx, uuid)
	}
}

func (m MeshTenantV4Client) Create(_ context.Context, tenant *client.MeshTenantV4Create) (*client.MeshTenantV4, error) {
	id := uuid.NewString()

	// Simulate a successful tenant creation with platformTenantId set
	platformTenantId := acctest.RandString(16)
	tenantName := tenant.Metadata.OwnedByWorkspace + "." + tenant.Metadata.OwnedByProject + "." + tenant.Spec.PlatformIdentifier

	created := &client.MeshTenantV4{
		Metadata: client.MeshTenantV4Metadata{
			Uuid:             id,
			OwnedByProject:   tenant.Metadata.OwnedByProject,
			OwnedByWorkspace: tenant.Metadata.OwnedByWorkspace,
			CreatedOn:        time.Now().UTC().Format(time.RFC3339),
		},
		Spec: client.MeshTenantV4Spec{
			PlatformIdentifier:    tenant.Spec.PlatformIdentifier,
			PlatformTenantId:      &platformTenantId,
			LandingZoneIdentifier: tenant.Spec.LandingZoneIdentifier,
			Quotas:                tenant.Spec.Quotas,
		},
		Status: client.MeshTenantV4Status{
			TenantName:             tenantName,
			PlatformTypeIdentifier: "mock-platform-type",
			Tags:                   map[string][]string{},
		},
	}

	m.Store[id] = created
	return created, nil
}

func (m MeshTenantV4Client) Delete(_ context.Context, uuid string) error {
	delete(m.Store, uuid)
	return nil
}

func (m MeshTenantV4Client) List(_ context.Context, query *client.MeshTenantV4Query) ([]client.MeshTenantV4, error) {
	var result []client.MeshTenantV4
	for _, t := range m.Store {
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
