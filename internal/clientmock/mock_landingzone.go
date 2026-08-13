package clientmock

import (
	"context"
	"fmt"

	"github.com/meshcloud/terraform-provider-meshstack/client"
)

type MeshLandingZoneClient struct {
	Store *Store[client.MeshLandingZone]
}

func (m MeshLandingZoneClient) Read(_ context.Context, name string) (*client.MeshLandingZone, error) {
	v, _ := m.Store.Get(name)
	return v, nil
}

// List applies only the plain attribute filters from the query; it does not simulate marketplace
// visibility or permissions. The mock stores the landing zone's platform as a uuid ref, so
// platform_uuid filters against that ref uuid.
func (m MeshLandingZoneClient) List(_ context.Context, query client.MeshLandingZoneListQuery) ([]client.MeshLandingZone, error) {
	var result []client.MeshLandingZone
	for _, landingZone := range m.Store.Values() {
		if query.PlatformUuid != nil && landingZone.Spec.PlatformRef.Uuid != *query.PlatformUuid {
			continue
		}
		if query.Identifier != nil && landingZone.Metadata.Name != *query.Identifier {
			continue
		}
		if query.DisplayName != nil && landingZone.Spec.DisplayName != *query.DisplayName {
			continue
		}
		if query.Restricted != nil && landingZone.Status.Restricted != *query.Restricted {
			continue
		}
		if query.OwnedByWorkspace != nil && landingZone.Metadata.OwnedByWorkspace != *query.OwnedByWorkspace {
			continue
		}
		result = append(result, *landingZone)
	}
	return result, nil
}

func (m MeshLandingZoneClient) Create(_ context.Context, landingZone *client.MeshLandingZoneCreate) (*client.MeshLandingZone, error) {
	spec := landingZone.Spec
	spec.Restricted = new(restrictedOr(spec.Restricted, nil))

	created := &client.MeshLandingZone{
		Metadata: landingZone.Metadata,
		Spec:     spec,
		Status: client.MeshLandingZoneStatus{
			Disabled:   false,
			Restricted: *spec.Restricted,
		},
	}
	m.Store.Set(landingZone.Metadata.Name, created)
	return created, nil
}

func (m MeshLandingZoneClient) Update(_ context.Context, name string, landingZone *client.MeshLandingZoneCreate) (*client.MeshLandingZone, error) {
	existing, _ := m.Store.Get(name)
	if existing == nil {
		return nil, fmt.Errorf("landing zone not found: %s", name)
	}

	spec := landingZone.Spec
	spec.Restricted = new(restrictedOr(spec.Restricted, existing.Spec.Restricted))

	existing.Metadata = landingZone.Metadata
	existing.Spec = spec
	existing.Status.Restricted = *spec.Restricted
	return existing, nil
}

// restrictedOr mirrors the backend: an omitted spec.restricted keeps the stored value rather than
// resetting it, so a landing zone restricted out of band survives an update that ignores the field.
func restrictedOr(requested, stored *bool) bool {
	if requested != nil {
		return *requested
	}
	if stored != nil {
		return *stored
	}
	return false
}

func (m MeshLandingZoneClient) Delete(_ context.Context, name string) error {
	m.Store.Delete(name)
	return nil
}
