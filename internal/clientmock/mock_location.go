package clientmock

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	"github.com/meshcloud/terraform-provider-meshstack/client"
)

type MeshLocationClient struct {
	Store *Store[client.MeshLocation]
}

func (m MeshLocationClient) Read(_ context.Context, name string) (*client.MeshLocation, error) {
	if location, ok := m.Store.Get(name); ok {
		return location, nil
	}
	return nil, nil
}

func (m MeshLocationClient) Create(_ context.Context, location *client.MeshLocationCreate) (*client.MeshLocation, error) {
	locationUuid := uuid.NewString()
	created := &client.MeshLocation{
		Metadata: client.MeshLocationMetadata{
			Name:             location.Metadata.Name,
			OwnedByWorkspace: location.Metadata.OwnedByWorkspace,
			Uuid:             locationUuid,
		},
		Spec: location.Spec,
		Status: client.MeshLocationStatus{
			IsPublic: false,
		},
	}
	m.Store.Set(created.Metadata.Name, created)
	return created, nil
}

func (m MeshLocationClient) Update(_ context.Context, name string, location *client.MeshLocationCreate) (*client.MeshLocation, error) {
	if existing, ok := m.Store.Get(name); ok {
		existing.Spec = location.Spec
		return existing, nil
	}
	return nil, fmt.Errorf("location not found: %s", name)
}

func (m MeshLocationClient) Delete(_ context.Context, name string) error {
	m.Store.Delete(name)
	return nil
}
