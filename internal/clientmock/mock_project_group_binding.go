package clientmock

import (
	"context"

	"github.com/meshcloud/terraform-provider-meshstack/client"
)

type MeshProjectGroupBindingClient struct {
	Store Store[client.MeshProjectGroupBinding]
}

func (m MeshProjectGroupBindingClient) Read(_ context.Context, name string) (*client.MeshProjectGroupBinding, error) {
	return m.Store[name], nil
}

func (m MeshProjectGroupBindingClient) Create(_ context.Context, binding *client.MeshProjectGroupBinding) (*client.MeshProjectGroupBinding, error) {
	m.Store[binding.Metadata.Name] = binding
	return binding, nil
}

func (m MeshProjectGroupBindingClient) Delete(_ context.Context, name string) error {
	delete(m.Store, name)
	return nil
}
