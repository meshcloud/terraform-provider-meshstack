package clientmock

import (
	"context"

	"github.com/meshcloud/meshstack-cli/client"
)

type MeshWorkspaceUserBindingClient struct {
	Store *Store[client.MeshWorkspaceUserBinding]
}

func (m MeshWorkspaceUserBindingClient) Read(_ context.Context, name string) (*client.MeshWorkspaceUserBinding, error) {
	v, _ := m.Store.Get(name)
	return v, nil
}

func (m MeshWorkspaceUserBindingClient) List(_ context.Context, filter client.MeshWorkspaceBindingListFilter) ([]client.MeshWorkspaceUserBinding, error) {
	var result []client.MeshWorkspaceUserBinding
	for _, binding := range m.Store.Values() {
		if matchesWorkspaceBindingFilter(binding.MeshWorkspaceBinding, filter) {
			result = append(result, *binding)
		}
	}
	return result, nil
}

func (m MeshWorkspaceUserBindingClient) Create(_ context.Context, binding *client.MeshWorkspaceUserBinding) (*client.MeshWorkspaceUserBinding, error) {
	binding.Status = &client.MeshUserBindingStatus{UserDisabled: false}
	m.Store.Set(binding.Metadata.Name, binding)
	return binding, nil
}

func (m MeshWorkspaceUserBindingClient) Delete(_ context.Context, name string) error {
	m.Store.Delete(name)
	return nil
}

func matchesWorkspaceBindingFilter(binding client.MeshWorkspaceBinding, filter client.MeshWorkspaceBindingListFilter) bool {
	return (filter.WorkspaceIdentifier == nil || binding.TargetRef.Name == *filter.WorkspaceIdentifier) &&
		(filter.WorkspaceRole == nil || binding.RoleRef.Name == *filter.WorkspaceRole) &&
		(filter.Username == nil || binding.Subject.Name == *filter.Username)
}
