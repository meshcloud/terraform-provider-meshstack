package clientmock

import (
	"context"

	"github.com/meshcloud/terraform-provider-meshstack/client"
)

type MeshWorkspaceGroupBindingClient struct {
	Store *Store[client.MeshWorkspaceGroupBinding]
}

func (m MeshWorkspaceGroupBindingClient) Read(_ context.Context, name string) (*client.MeshWorkspaceGroupBinding, error) {
	v, _ := m.Store.Get(name)
	return v, nil
}

func (m MeshWorkspaceGroupBindingClient) Create(_ context.Context, binding *client.MeshWorkspaceGroupBinding) (*client.MeshWorkspaceGroupBinding, error) {
	m.Store.Set(binding.Metadata.Name, binding)
	return binding, nil
}

func (m MeshWorkspaceGroupBindingClient) Delete(_ context.Context, name string) error {
	m.Store.Delete(name)
	return nil
}
