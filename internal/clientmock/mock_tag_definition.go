package clientmock

import (
	"context"
	"fmt"

	"github.com/meshcloud/meshstack-cli/client"
)

type MeshTagDefinitionClient struct {
	client.MeshTagDefinitionClient

	Store *Store[client.MeshTagDefinition]
}

func (m MeshTagDefinitionClient) List(context.Context, client.NoFilter, ...client.ListOption) ([]client.MeshTagDefinition, error) {
	var result []client.MeshTagDefinition
	for _, def := range m.Store.Values() {
		result = append(result, *def)
	}
	return result, nil
}

func (m MeshTagDefinitionClient) Read(_ context.Context, name string) (*client.MeshTagDefinition, error) {
	if def, ok := m.Store.Get(name); ok {
		return def, nil
	}
	return nil, nil
}

func (m MeshTagDefinitionClient) Create(_ context.Context, tagDefinition client.MeshTagDefinition) (client.MeshTagDefinition, error) {
	created := &client.MeshTagDefinition{
		Metadata: tagDefinition.Metadata,
		Spec:     tagDefinition.Spec,
	}
	m.Store.Set(created.Metadata.Name, created)
	return *created, nil
}

func (m MeshTagDefinitionClient) Update(_ context.Context, tagDefinition client.MeshTagDefinition) (client.MeshTagDefinition, error) {
	name := tagDefinition.Metadata.Name
	if existing, ok := m.Store.Get(name); ok {
		existing.Spec = tagDefinition.Spec
		return *existing, nil
	}
	return client.MeshTagDefinition{}, fmt.Errorf("tag definition not found: %s", name)
}

func (m MeshTagDefinitionClient) Delete(_ context.Context, name string) error {
	m.Store.Delete(name)
	return nil
}
