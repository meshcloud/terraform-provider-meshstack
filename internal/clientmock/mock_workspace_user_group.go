package clientmock

import (
	"context"
	"fmt"
	"slices"

	"github.com/meshcloud/meshstack-cli/client"
)

type MeshWorkspaceUserGroupClient struct {
	Store     *Store[client.MeshWorkspaceUserGroup]
	UserStore *Store[client.MeshUser]
}

func (m MeshWorkspaceUserGroupClient) List(_ context.Context, filter client.MeshWorkspaceUserGroupListFilter) ([]client.MeshWorkspaceUserGroup, error) {
	var result []client.MeshWorkspaceUserGroup
	for _, identifier := range m.Store.SortedKeys() {
		group, _ := m.Store.Get(identifier)
		if (filter.WorkspaceIdentifier == "" || group.Metadata.OwnedByWorkspace == filter.WorkspaceIdentifier) &&
			(filter.Username == "" || slices.Contains(group.Spec.Members, filter.Username)) &&
			(filter.SourceSystem == "" || group.Metadata.SourceSystem != nil && *group.Metadata.SourceSystem == filter.SourceSystem) {
			result = append(result, *group)
		}
	}
	return result, nil
}

func (m MeshWorkspaceUserGroupClient) Read(_ context.Context, ownedByWorkspace, name string) (*client.MeshWorkspaceUserGroup, error) {
	group, ok := m.Store.Get(client.MeshWorkspaceUserGroupIdentifier(ownedByWorkspace, name))
	if !ok {
		return nil, nil
	}
	copied := *group
	copied.Spec.Members = slices.Clone(group.Spec.Members)
	return &copied, nil
}

func (m MeshWorkspaceUserGroupClient) Create(_ context.Context, group *client.MeshWorkspaceUserGroup) (*client.MeshWorkspaceUserGroup, error) {
	identifier := group.Metadata.Identifier()
	if _, exists := m.Store.Get(identifier); exists {
		return nil, fmt.Errorf("meshWorkspaceUserGroup %s already exists", identifier)
	}
	return m.store(group)
}

func (m MeshWorkspaceUserGroupClient) Update(_ context.Context, group *client.MeshWorkspaceUserGroup) (*client.MeshWorkspaceUserGroup, error) {
	existing, ok := m.Store.Get(group.Metadata.Identifier())
	if !ok {
		return nil, fmt.Errorf("meshWorkspaceUserGroup %s not found", group.Metadata.Identifier())
	}
	if group.Metadata.SourceSystem != nil && !equalPointers(existing.Metadata.SourceSystem, group.Metadata.SourceSystem) {
		return nil, fmt.Errorf("metadata.sourceSystem of meshWorkspaceUserGroup %s cannot change", group.Metadata.Identifier())
	}
	group.Metadata.SourceSystem = existing.Metadata.SourceSystem
	return m.store(group)
}

func (m MeshWorkspaceUserGroupClient) Delete(_ context.Context, ownedByWorkspace, name string) error {
	m.Store.Delete(client.MeshWorkspaceUserGroupIdentifier(ownedByWorkspace, name))
	return nil
}

func (m MeshWorkspaceUserGroupClient) store(group *client.MeshWorkspaceUserGroup) (*client.MeshWorkspaceUserGroup, error) {
	disabledUsernames := []string{}
	for _, username := range group.Spec.Members {
		user, ok := m.UserStore.Get(username)
		if !ok {
			return nil, fmt.Errorf("member %q of meshWorkspaceUserGroup %s is not a known meshUser", username, group.Metadata.Identifier())
		}
		if user.Status.Lifecycle.State == "DEACTIVATED" {
			disabledUsernames = append(disabledUsernames, username)
		}
	}
	stored := *group
	stored.Spec.Members = slices.Clone(group.Spec.Members)
	if stored.Spec.Members == nil {
		stored.Spec.Members = []string{}
	}
	if stored.Spec.Tags == nil {
		stored.Spec.Tags = map[string][]string{}
	}
	stored.Status = &client.MeshWorkspaceUserGroupStatus{DisabledUsernames: disabledUsernames}
	m.Store.Set(stored.Metadata.Identifier(), &stored)
	result := stored
	result.Spec.Members = slices.Clone(stored.Spec.Members)
	return &result, nil
}

func equalPointers[T comparable](a, b *T) bool {
	return a == nil && b == nil || a != nil && b != nil && *a == *b
}
