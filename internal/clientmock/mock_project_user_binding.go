package clientmock

import (
	"context"
	"iter"

	"github.com/meshcloud/meshstack-cli/client"
)

type MeshProjectUserBindingClient struct {
	client.MeshProjectUserBindingClient

	Store *Store[client.MeshProjectUserBinding]
}

func (m MeshProjectUserBindingClient) Read(_ context.Context, name string) (*client.MeshProjectUserBinding, error) {
	v, _ := m.Store.Get(name)
	return v, nil
}

func (m MeshProjectUserBindingClient) List(_ context.Context, filter client.MeshProjectBindingFilter, _ ...client.ListOption) iter.Seq2[client.MeshProjectUserBinding, error] {
	var result []client.MeshProjectUserBinding
	for _, binding := range m.Store.Values() {
		if matchesProjectBindingFilter(binding.MeshProjectBinding, filter) {
			result = append(result, *binding)
		}
	}
	return listOf(result, nil)
}

func (m MeshProjectUserBindingClient) Create(_ context.Context, binding client.MeshProjectUserBinding) (client.MeshProjectUserBinding, error) {
	binding.Status = &client.MeshUserBindingStatus{UserDisabled: false}
	m.Store.Set(binding.Metadata.Name, &binding)
	return binding, nil
}

func (m MeshProjectUserBindingClient) Delete(_ context.Context, name string) error {
	m.Store.Delete(name)
	return nil
}

func matchesProjectBindingFilter(binding client.MeshProjectBinding, filter client.MeshProjectBindingFilter) bool {
	return (filter.WorkspaceIdentifier == nil || binding.TargetRef.OwnedByWorkspace == *filter.WorkspaceIdentifier) &&
		(filter.ProjectIdentifier == nil || binding.TargetRef.Name == *filter.ProjectIdentifier)
}
