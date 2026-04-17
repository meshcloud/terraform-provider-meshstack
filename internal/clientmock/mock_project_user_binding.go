package clientmock

import (
	"context"

	"github.com/meshcloud/terraform-provider-meshstack/client"
)

type MeshProjectUserBindingClient struct {
	Store *Store[client.MeshProjectUserBinding]
}

func (m MeshProjectUserBindingClient) Read(_ context.Context, name string) (*client.MeshProjectUserBinding, error) {
	v, _ := m.Store.Get(name)
	return v, nil
}

func (m MeshProjectUserBindingClient) Create(_ context.Context, binding *client.MeshProjectUserBinding) (*client.MeshProjectUserBinding, error) {
	m.Store.Set(binding.Metadata.Name, binding)
	return binding, nil
}

func (m MeshProjectUserBindingClient) Delete(_ context.Context, name string) error {
	m.Store.Delete(name)
	return nil
}
