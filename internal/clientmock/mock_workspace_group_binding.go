package clientmock

import (
	"context"
	"errors"

	"github.com/meshcloud/meshstack-cli/client"
)

type MeshWorkspaceGroupBindingClient struct {
	Store *Store[client.MeshWorkspaceGroupBinding]
}

func (m MeshWorkspaceGroupBindingClient) Read(_ context.Context, name string) (*client.MeshWorkspaceGroupBinding, error) {
	v, _ := m.Store.Get(name)
	return v, nil
}

func (m MeshWorkspaceGroupBindingClient) List(_ context.Context, filter client.MeshWorkspaceBindingListFilter) ([]client.MeshWorkspaceGroupBinding, error) {
	if filter.Username != nil {
		return nil, errors.New("group membership is not mocked")
	}
	var result []client.MeshWorkspaceGroupBinding
	for _, binding := range m.Store.Values() {
		if matchesWorkspaceBindingFilter(binding.MeshWorkspaceBinding, filter) {
			result = append(result, *binding)
		}
	}
	return result, nil
}

func (m MeshWorkspaceGroupBindingClient) Create(_ context.Context, binding *client.MeshWorkspaceGroupBinding) (*client.MeshWorkspaceGroupBinding, error) {
	m.Store.Set(binding.Metadata.Name, binding)
	return binding, nil
}

func (m MeshWorkspaceGroupBindingClient) Delete(_ context.Context, name string) error {
	m.Store.Delete(name)
	return nil
}
