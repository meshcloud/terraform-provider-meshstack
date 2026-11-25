package clientmock

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/meshcloud/meshstack-cli/client"
)

type MeshProjectClient struct {
	client.MeshProjectClient

	Store *Store[client.MeshProject]
}

func (m MeshProjectClient) Read(_ context.Context, id string) (*client.MeshProject, error) {
	v, _ := m.Store.Get(id)
	return copyProject(v), nil
}

func (m MeshProjectClient) List(_ context.Context, filter client.MeshProjectFilter, _ ...client.ListOption) ([]client.MeshProject, error) {
	var wantedPaymentMethod *string
	if filter.PaymentIdentifier != nil {
		wantedPaymentMethod = new(*filter.PaymentIdentifier)
		if strings.TrimSpace(*wantedPaymentMethod) == "" {
			*wantedPaymentMethod = ""
		}
	}
	var result []client.MeshProject
	for _, p := range m.Store.Values() {
		if p.Metadata.OwnedByWorkspace != filter.WorkspaceIdentifier {
			continue
		}
		if wantedPaymentMethod != nil {
			projectPaymentMethod := ""
			if p.Spec.PaymentMethodIdentifier != nil {
				projectPaymentMethod = *p.Spec.PaymentMethodIdentifier
			}
			if projectPaymentMethod != *wantedPaymentMethod {
				continue
			}
		}
		result = append(result, *p)
	}
	return result, nil
}

func (m MeshProjectClient) Create(_ context.Context, project client.MeshProject) (client.MeshProject, error) {
	created := &client.MeshProject{
		Metadata: client.MeshProjectMetadata{
			Name:             project.Metadata.Name,
			OwnedByWorkspace: project.Metadata.OwnedByWorkspace,
			CreatedOn:        time.Now().UTC().Format(time.RFC3339),
		},
		Spec: project.Spec,
	}
	created.Spec.Tags = copyTags(project.Spec.Tags)
	m.Store.Set(client.NewMeshProjectId(project.Metadata.OwnedByWorkspace, project.Metadata.Name), created)
	return *copyProject(created), nil
}

func (m MeshProjectClient) Update(_ context.Context, project client.MeshProject) (client.MeshProject, error) {
	key := client.NewMeshProjectId(project.Metadata.OwnedByWorkspace, project.Metadata.Name)
	existing, _ := m.Store.Get(key)
	if existing == nil {
		return client.MeshProject{}, fmt.Errorf("project not found: %s", key)
	}
	existing.Spec = project.Spec
	existing.Spec.Tags = copyTags(project.Spec.Tags)
	return *copyProject(existing), nil
}

func (m MeshProjectClient) Delete(_ context.Context, id string) error {
	m.Store.Delete(id)
	return nil
}

func copyProject(project *client.MeshProject) *client.MeshProject {
	if project == nil {
		return nil
	}
	copied := *project
	copied.Spec.Tags = copyTags(project.Spec.Tags)
	return &copied
}
