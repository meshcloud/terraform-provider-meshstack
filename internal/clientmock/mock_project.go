package clientmock

import (
	"context"
	"fmt"
	"time"

	"github.com/meshcloud/terraform-provider-meshstack/client"
)

type MeshProjectClient struct {
	Store Store[client.MeshProject]
}

func (m MeshProjectClient) Read(_ context.Context, workspace string, name string) (*client.MeshProject, error) {
	return m.Store[workspace+"."+name], nil
}

func (m MeshProjectClient) List(_ context.Context, workspaceIdentifier string, paymentMethodIdentifier *string) ([]client.MeshProject, error) {
	var result []client.MeshProject
	for _, p := range m.Store {
		if p.Metadata.OwnedByWorkspace == workspaceIdentifier {
			result = append(result, *p)
		}
	}
	return result, nil
}

func (m MeshProjectClient) Create(_ context.Context, project *client.MeshProjectCreate) (*client.MeshProject, error) {
	created := &client.MeshProject{
		Metadata: client.MeshProjectMetadata{
			Name:             project.Metadata.Name,
			OwnedByWorkspace: project.Metadata.OwnedByWorkspace,
			CreatedOn:        time.Now().UTC().Format(time.RFC3339),
		},
		Spec: project.Spec,
	}
	m.Store[project.Metadata.OwnedByWorkspace+"."+project.Metadata.Name] = created
	return created, nil
}

func (m MeshProjectClient) Update(_ context.Context, project *client.MeshProjectCreate) (*client.MeshProject, error) {
	key := project.Metadata.OwnedByWorkspace + "." + project.Metadata.Name
	existing := m.Store[key]
	if existing == nil {
		return nil, fmt.Errorf("project not found: %s", key)
	}
	existing.Spec = project.Spec
	return existing, nil
}

func (m MeshProjectClient) Delete(_ context.Context, workspace string, name string) error {
	delete(m.Store, workspace+"."+name)
	return nil
}
