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
	if v == nil {
		return nil, nil
	}
	// Return a copy to avoid mutation side effects
	cp := *v
	if v.Metadata.Tags != nil {
		cp.Metadata.Tags = make(map[string][]string, len(v.Metadata.Tags))
		for k, val := range v.Metadata.Tags {
			cp.Metadata.Tags[k] = append([]string(nil), val...)
		}
	}
	return &cp, nil
}

func (m MeshWorkspaceClient) Create(_ context.Context, workspace *client.MeshWorkspaceCreate) (*client.MeshWorkspace, error) {
	tagsCopy := make(map[string][]string)
	if workspace.Metadata.Tags != nil {
		for k, v := range workspace.Metadata.Tags {
			tagsCopy[k] = append([]string(nil), v...)
		}
	}
	created := &client.MeshWorkspace{
		Metadata: client.MeshWorkspaceMetadata{
			Name:      workspace.Metadata.Name,
			CreatedOn: time.Now().UTC().Format(time.RFC3339),
			Tags:      tagsCopy,
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

	tagsCopy := make(map[string][]string)
	if workspace.Metadata.Tags != nil {
		for k, v := range workspace.Metadata.Tags {
			tagsCopy[k] = append([]string(nil), v...)
		}
	}

	updated := &client.MeshWorkspace{
		Metadata: client.MeshWorkspaceMetadata{
			Name:      existing.Metadata.Name,
			CreatedOn: existing.Metadata.CreatedOn,
			DeletedOn: existing.Metadata.DeletedOn,
			Tags:      tagsCopy,
		},
		Spec: workspace.Spec,
	}
	m.Store.Set(name, updated)
	return updated, nil
}

func (m MeshWorkspaceClient) Delete(_ context.Context, name string) error {
	m.Store.Delete(name)
	return nil
}
