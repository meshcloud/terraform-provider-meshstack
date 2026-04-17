package clientmock

import (
	"context"

	"github.com/meshcloud/terraform-provider-meshstack/client"
)

type MeshProjectGroupBindingClient struct {
	Store *Store[client.MeshProjectGroupBinding]
}

func (m MeshProjectGroupBindingClient) Read(_ context.Context, name string) (*client.MeshProjectGroupBinding, error) {
	v, _ := m.Store.Get(name)
	return v, nil
}

func (m MeshProjectGroupBindingClient) Create(_ context.Context, binding *client.MeshProjectGroupBinding) (*client.MeshProjectGroupBinding, error) {
	m.Store.Set(binding.Metadata.Name, binding)
	return binding, nil
}

func (m MeshProjectGroupBindingClient) Delete(_ context.Context, name string) error {
	m.Store.Delete(name)
	return nil
}
