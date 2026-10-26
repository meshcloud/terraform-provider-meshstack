package clientmock

import (
	"context"

	"github.com/meshcloud/meshstack-cli/client"
)

type MeshProjectGroupBindingClient struct {
	Store *Store[client.MeshProjectGroupBinding]
}

func (m MeshProjectGroupBindingClient) Read(_ context.Context, name string) (*client.MeshProjectGroupBinding, error) {
	v, _ := m.Store.Get(name)
	return v, nil
}

func (m MeshProjectGroupBindingClient) List(_ context.Context, filter client.MeshProjectBindingListFilter) ([]client.MeshProjectGroupBinding, error) {
	var result []client.MeshProjectGroupBinding
	for _, binding := range m.Store.Values() {
		if matchesProjectBindingFilter(binding.MeshProjectBinding, filter) {
			result = append(result, *binding)
		}
	}
	return result, nil
}

func (m MeshProjectGroupBindingClient) Create(_ context.Context, binding *client.MeshProjectGroupBinding) (*client.MeshProjectGroupBinding, error) {
	m.Store.Set(binding.Metadata.Name, binding)
	return binding, nil
}

func (m MeshProjectGroupBindingClient) Delete(_ context.Context, name string) error {
	m.Store.Delete(name)
	return nil
}
