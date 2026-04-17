package clientmock

import (
	"context"

	"github.com/meshcloud/terraform-provider-meshstack/client"
)

type MeshWorkspaceUserBindingClient struct {
	Store *Store[client.MeshWorkspaceUserBinding]
}

func (m MeshWorkspaceUserBindingClient) Read(_ context.Context, name string) (*client.MeshWorkspaceUserBinding, error) {
	v, _ := m.Store.Get(name)
	return v, nil
}

func (m MeshWorkspaceUserBindingClient) Create(_ context.Context, binding *client.MeshWorkspaceUserBinding) (*client.MeshWorkspaceUserBinding, error) {
	m.Store.Set(binding.Metadata.Name, binding)
	return binding, nil
}

func (m MeshWorkspaceUserBindingClient) Delete(_ context.Context, name string) error {
	m.Store.Delete(name)
	return nil
}
