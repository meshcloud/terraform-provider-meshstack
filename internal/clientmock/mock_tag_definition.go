package clientmock

import (
	"context"
	"fmt"
	"sync"

	"github.com/meshcloud/terraform-provider-meshstack/client"
)

var meshTagDefinitionStoreMu sync.RWMutex

type MeshTagDefinitionClient struct {
	Store Store[client.MeshTagDefinition]
}

func (m MeshTagDefinitionClient) List(_ context.Context) ([]client.MeshTagDefinition, error) {
	meshTagDefinitionStoreMu.RLock()
	defer meshTagDefinitionStoreMu.RUnlock()

	var result []client.MeshTagDefinition
	for _, def := range m.Store {
		result = append(result, *def)
	}
	return result, nil
}

func (m MeshTagDefinitionClient) Read(_ context.Context, name string) (*client.MeshTagDefinition, error) {
	meshTagDefinitionStoreMu.RLock()
	defer meshTagDefinitionStoreMu.RUnlock()

	if def, ok := m.Store[name]; ok {
		return def, nil
	}
	return nil, nil
}

func (m MeshTagDefinitionClient) Create(_ context.Context, tagDefinition *client.MeshTagDefinition) (*client.MeshTagDefinition, error) {
	meshTagDefinitionStoreMu.Lock()
	defer meshTagDefinitionStoreMu.Unlock()

	created := &client.MeshTagDefinition{
		Metadata: tagDefinition.Metadata,
		Spec:     tagDefinition.Spec,
	}
	m.Store[created.Metadata.Name] = created
	return created, nil
}

func (m MeshTagDefinitionClient) Update(_ context.Context, tagDefinition *client.MeshTagDefinition) (*client.MeshTagDefinition, error) {
	meshTagDefinitionStoreMu.Lock()
	defer meshTagDefinitionStoreMu.Unlock()

	name := tagDefinition.Metadata.Name
	if existing, ok := m.Store[name]; ok {
		existing.Spec = tagDefinition.Spec
		return existing, nil
	}
	return nil, fmt.Errorf("tag definition not found: %s", name)
}

func (m MeshTagDefinitionClient) Delete(_ context.Context, name string) error {
	meshTagDefinitionStoreMu.Lock()
	defer meshTagDefinitionStoreMu.Unlock()

	delete(m.Store, name)
	return nil
}
