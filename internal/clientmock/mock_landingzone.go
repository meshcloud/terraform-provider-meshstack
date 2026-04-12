package clientmock

import (
	"context"
	"fmt"

	"github.com/meshcloud/terraform-provider-meshstack/client"
)

type MeshLandingZoneClient struct {
	Store Store[client.MeshLandingZone]
}

func (m MeshLandingZoneClient) Read(_ context.Context, name string) (*client.MeshLandingZone, error) {
	return m.Store[name], nil
}

func (m MeshLandingZoneClient) Create(_ context.Context, landingZone *client.MeshLandingZoneCreate) (*client.MeshLandingZone, error) {
	created := &client.MeshLandingZone{
		Metadata: landingZone.Metadata,
		Spec:     landingZone.Spec,
		Status: client.MeshLandingZoneStatus{
			Disabled:   false,
			Restricted: false,
		},
	}
	m.Store[landingZone.Metadata.Name] = created
	return created, nil
}

func (m MeshLandingZoneClient) Update(_ context.Context, name string, landingZone *client.MeshLandingZoneCreate) (*client.MeshLandingZone, error) {
	existing := m.Store[name]
	if existing == nil {
		return nil, fmt.Errorf("landing zone not found: %s", name)
	}
	existing.Metadata = landingZone.Metadata
	existing.Spec = landingZone.Spec
	return existing, nil
}

func (m MeshLandingZoneClient) Delete(_ context.Context, name string) error {
	delete(m.Store, name)
	return nil
}
