package clientmock

import (
	"context"

	"github.com/meshcloud/terraform-provider-meshstack/client"
)

type MeshProjectUserBindingClient struct {
	Store Store[client.MeshProjectUserBinding]
}

func (m MeshProjectUserBindingClient) Read(_ context.Context, name string) (*client.MeshProjectUserBinding, error) {
	return m.Store[name], nil
}

func (m MeshProjectUserBindingClient) Create(_ context.Context, binding *client.MeshProjectUserBinding) (*client.MeshProjectUserBinding, error) {
	m.Store[binding.Metadata.Name] = binding
	return binding, nil
}

func (m MeshProjectUserBindingClient) Delete(_ context.Context, name string) error {
	delete(m.Store, name)
	return nil
}
