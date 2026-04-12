package clientmock

import (
	"context"
	"fmt"

	"github.com/meshcloud/terraform-provider-meshstack/client"
)

type MeshTenantClient struct {
	Store Store[client.MeshTenant]
}

func (m MeshTenantClient) tenantId(workspace, project, platform string) string {
	return workspace + "." + project + "." + platform
}

func (m MeshTenantClient) Read(_ context.Context, workspace string, project string, platform string) (*client.MeshTenant, error) {
	return m.Store[m.tenantId(workspace, project, platform)], nil
}

func (m MeshTenantClient) Create(_ context.Context, tenant *client.MeshTenantCreate) (*client.MeshTenant, error) {
	id := m.tenantId(tenant.Metadata.OwnedByWorkspace, tenant.Metadata.OwnedByProject, tenant.Metadata.PlatformIdentifier)

	if m.Store[id] != nil {
		return nil, fmt.Errorf("tenant already exists: %s", id)
	}

	created := &client.MeshTenant{
		Metadata: client.MeshTenantMetadata{
			OwnedByProject:     tenant.Metadata.OwnedByProject,
			OwnedByWorkspace:   tenant.Metadata.OwnedByWorkspace,
			PlatformIdentifier: tenant.Metadata.PlatformIdentifier,
			AssignedTags:       map[string][]string{},
		},
		Spec: client.MeshTenantSpec{
			LocalId:               tenant.Spec.LocalId,
			LandingZoneIdentifier: *tenant.Spec.LandingZoneIdentifier,
		},
	}

	m.Store[id] = created
	return created, nil
}

func (m MeshTenantClient) Delete(_ context.Context, workspace string, project string, platform string) error {
	delete(m.Store, m.tenantId(workspace, project, platform))
	return nil
}
