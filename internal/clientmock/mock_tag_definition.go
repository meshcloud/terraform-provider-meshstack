package clientmock

import (
	"context"
	"fmt"

	"github.com/meshcloud/terraform-provider-meshstack/client"
)

type MeshTagDefinitionClient struct {
	Store Store[client.MeshTagDefinition]
}

func (m MeshTagDefinitionClient) List(_ context.Context) ([]client.MeshTagDefinition, error) {
	var result []client.MeshTagDefinition
	for _, def := range m.Store {
		result = append(result, *def)
	}
	return result, nil
}

func (m MeshTagDefinitionClient) Read(_ context.Context, name string) (*client.MeshTagDefinition, error) {
	if def, ok := m.Store[name]; ok {
		return def, nil
	}
	return nil, nil
}

func (m MeshTagDefinitionClient) Create(_ context.Context, tagDefinition *client.MeshTagDefinition) (*client.MeshTagDefinition, error) {
	created := &client.MeshTagDefinition{
		ApiVersion: client.API_VERSION_TAG_DEFINITION,
		Kind:       "meshTagDefinition",
		Metadata:   tagDefinition.Metadata,
		Spec:       tagDefinition.Spec,
	}
	m.Store[created.Metadata.Name] = created
	return created, nil
}

func (m MeshTagDefinitionClient) Update(_ context.Context, tagDefinition *client.MeshTagDefinition) (*client.MeshTagDefinition, error) {
	name := tagDefinition.Metadata.Name
	if existing, ok := m.Store[name]; ok {
		existing.Spec = tagDefinition.Spec
		return existing, nil
	}
	return nil, fmt.Errorf("tag definition not found: %s", name)
}

func (m MeshTagDefinitionClient) Delete(_ context.Context, name string) error {
	delete(m.Store, name)
	return nil
}
