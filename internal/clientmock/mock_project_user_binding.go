package clientmock

import (
	"context"

	"github.com/meshcloud/meshstack-cli/client"
)

type MeshProjectUserBindingClient struct {
	Store *Store[client.MeshProjectUserBinding]
}

func (m MeshProjectUserBindingClient) Read(_ context.Context, name string) (*client.MeshProjectUserBinding, error) {
	v, _ := m.Store.Get(name)
	return v, nil
}

func (m MeshProjectUserBindingClient) List(_ context.Context, filter client.MeshProjectBindingListFilter) ([]client.MeshProjectUserBinding, error) {
	var result []client.MeshProjectUserBinding
	for _, binding := range m.Store.Values() {
		if matchesProjectBindingFilter(binding.MeshProjectBinding, filter) {
			result = append(result, *binding)
		}
	}
	return result, nil
}

func (m MeshProjectUserBindingClient) Create(_ context.Context, binding *client.MeshProjectUserBinding) (*client.MeshProjectUserBinding, error) {
	m.Store.Set(binding.Metadata.Name, binding)
	return binding, nil
}

func (m MeshProjectUserBindingClient) Delete(_ context.Context, name string) error {
	m.Store.Delete(name)
	return nil
}

func matchesProjectBindingFilter(binding client.MeshProjectBinding, filter client.MeshProjectBindingListFilter) bool {
	return (filter.WorkspaceIdentifier == nil || binding.TargetRef.OwnedByWorkspace == *filter.WorkspaceIdentifier) &&
		(filter.ProjectIdentifier == nil || binding.TargetRef.Name == *filter.ProjectIdentifier)
}
