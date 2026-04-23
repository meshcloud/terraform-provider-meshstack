package clientmock

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	"github.com/meshcloud/terraform-provider-meshstack/client"
)

type MeshPlatformTypeClient struct {
	Store *Store[client.MeshPlatformType]
}

func (m MeshPlatformTypeClient) Read(_ context.Context, identifier string) (*client.MeshPlatformType, error) {
	if platformType, ok := m.Store.Get(identifier); ok {
		return platformType, nil
	}
	return nil, nil
}

func (m MeshPlatformTypeClient) Create(_ context.Context, platformType *client.MeshPlatformTypeCreate) (*client.MeshPlatformType, error) {
	platformTypeUuid := uuid.NewString()
	created := &client.MeshPlatformType{
		Metadata: client.MeshPlatformTypeMetadata{
			Name:             platformType.Metadata.Name,
			OwnedByWorkspace: platformType.Metadata.OwnedByWorkspace,
			Uuid:             new(platformTypeUuid),
		},
		Spec: platformType.Spec,
		Status: client.MeshPlatformTypeStatus{
			Lifecycle: client.MeshPlatformTypeLifecycle{
				State: "ACTIVE",
			},
		},
	}
	m.Store.Set(created.Metadata.Name, created)
	return created, nil
}

func (m MeshPlatformTypeClient) Update(_ context.Context, name string, platformType *client.MeshPlatformTypeCreate) (*client.MeshPlatformType, error) {
	if existing, ok := m.Store.Get(name); ok {
		existing.Spec = platformType.Spec
		return existing, nil
	}
	return nil, fmt.Errorf("platform type not found: %s", name)
}

func (m MeshPlatformTypeClient) Delete(_ context.Context, name string) error {
	m.Store.Delete(name)
	return nil
}

func (m MeshPlatformTypeClient) List(_ context.Context, category *string, lifecycleStatus *string) ([]client.MeshPlatformType, error) {
	var result []client.MeshPlatformType
	for _, platformType := range m.Store.Values() {
		if category != nil && platformType.Spec.Category != *category {
			continue
		}
		if lifecycleStatus != nil && platformType.Status.Lifecycle.State != *lifecycleStatus {
			continue
		}
		result = append(result, *platformType)
	}
	return result, nil
}
