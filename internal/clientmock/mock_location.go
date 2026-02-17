package clientmock

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"

	"github.com/meshcloud/terraform-provider-meshstack/client"
)

type MeshLocationClient struct {
	Store Store[client.MeshLocation]
}

func (m MeshLocationClient) Read(_ context.Context, name string) (*client.MeshLocation, error) {
	if location, ok := m.Store[name]; ok {
		return location, nil
	}
	return nil, nil
}

func (m MeshLocationClient) Create(_ context.Context, location *client.MeshLocationCreate) (*client.MeshLocation, error) {
	locationUuid := acctest.RandString(32)
	created := &client.MeshLocation{
		ApiVersion: location.ApiVersion,
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
	m.Store[created.Metadata.Name] = created
	return created, nil
}

func (m MeshLocationClient) Update(_ context.Context, name string, location *client.MeshLocationCreate) (*client.MeshLocation, error) {
	if existing, ok := m.Store[name]; ok {
		existing.Spec = location.Spec
		return existing, nil
	}
	return nil, fmt.Errorf("location not found: %s", name)
}

func (m MeshLocationClient) Delete(_ context.Context, name string) error {
	delete(m.Store, name)
	return nil
}
