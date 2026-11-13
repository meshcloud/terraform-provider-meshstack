package clientmock

import (
	"context"
	"fmt"
	"iter"

	"github.com/google/uuid"
	"github.com/meshcloud/meshstack-cli/client"
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

func (m MeshPlatformTypeClient) Create(_ context.Context, platformType client.MeshPlatformType) (client.MeshPlatformType, error) {
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
	return *created, nil
}

func (m MeshPlatformTypeClient) Update(_ context.Context, platformType client.MeshPlatformType) (client.MeshPlatformType, error) {
	name := platformType.Metadata.Name
	if existing, ok := m.Store.Get(name); ok {
		existing.Spec = platformType.Spec
		return *existing, nil
	}
	return client.MeshPlatformType{}, fmt.Errorf("platform type not found: %s", name)
}

func (m MeshPlatformTypeClient) Delete(_ context.Context, name string) error {
	m.Store.Delete(name)
	return nil
}

func (m MeshPlatformTypeClient) List(_ context.Context, filter client.MeshPlatformTypeFilter, _ ...client.ListOption) iter.Seq2[client.MeshPlatformType, error] {
	var result []client.MeshPlatformType
	for _, platformType := range m.Store.Values() {
		if filter.Category != nil && platformType.Spec.Category != *filter.Category {
			continue
		}
		if filter.LifecycleStatus != nil && platformType.Status.Lifecycle.State != *filter.LifecycleStatus {
			continue
		}
		result = append(result, *platformType)
	}
	return listOf(result, nil)
}
