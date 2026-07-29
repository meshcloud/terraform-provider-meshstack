package clientmock

import (
	"context"
	"fmt"
	"slices"
	"time"

	"github.com/meshcloud/terraform-provider-meshstack/client"
)

type MeshWorkspaceClient struct {
	Store *Store[client.MeshWorkspace]
}

// copyTags deep-copies a tag map so stored values cannot be mutated through a returned DTO. It clones
// each value list with slices.Clone rather than append-to-nil, which would collapse a tag declared with
// no values to nil: the API sends `[]` for such a tag, and a nil list reaches the provider as a *null*
// list instead of an empty one — a diff no configuration can settle.
func copyTags(tags map[string][]string) map[string][]string {
	cp := make(map[string][]string, len(tags))
	for k, v := range tags {
		cp[k] = slices.Clone(v)
	}
	return cp
}

func (m MeshWorkspaceClient) Read(_ context.Context, name string) (*client.MeshWorkspace, error) {
	v, _ := m.Store.Get(name)
	if v == nil {
		return nil, nil
	}
	// Return a copy to avoid mutation side effects
	cp := *v
	if v.Metadata.Tags != nil {
		cp.Metadata.Tags = copyTags(v.Metadata.Tags)
	}
	return &cp, nil
}

func (m MeshWorkspaceClient) Create(_ context.Context, workspace *client.MeshWorkspaceCreate) (*client.MeshWorkspace, error) {
	tagsCopy := copyTags(workspace.Metadata.Tags)
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

	tagsCopy := copyTags(workspace.Metadata.Tags)
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
