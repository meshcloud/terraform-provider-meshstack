package clientmock

import (
	"context"
	"fmt"
	"time"

	"github.com/meshcloud/terraform-provider-meshstack/client"
)

type MeshWorkspaceClient struct {
	Store *Store[client.MeshWorkspace]
}

func (m MeshWorkspaceClient) Read(_ context.Context, name string) (*client.MeshWorkspace, error) {
	v, _ := m.Store.Get(name)
	return v, nil
}

func (m MeshWorkspaceClient) Create(_ context.Context, workspace *client.MeshWorkspaceCreate) (*client.MeshWorkspace, error) {
	created := &client.MeshWorkspace{
		Metadata: client.MeshWorkspaceMetadata{
			Name:      workspace.Metadata.Name,
			CreatedOn: time.Now().UTC().Format(time.RFC3339),
			Tags:      workspace.Metadata.Tags,
		},
		Spec: workspace.Spec,
	}

	m.Store.Set(workspace.Metadata.Name, created)
	return created, nil
}

func (m MeshWorkspaceClient) Update(_ context.Context, name string, workspace *client.MeshWorkspaceCreate) (*client.MeshWorkspace, error) {
	existing, _ := m.Store.Get(name)
	if existing == nil {
		return nil, fmt.Errorf("workspace not found: %s", name)
	}

	existing.Spec = workspace.Spec
	existing.Metadata.Tags = workspace.Metadata.Tags
	return existing, nil
}

func (m MeshWorkspaceClient) Delete(_ context.Context, name string) error {
	m.Store.Delete(name)
	return nil
}
