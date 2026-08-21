package clientmock

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	"github.com/meshcloud/terraform-provider-meshstack/client"
)

type MeshIntegrationClient struct {
	Store *Store[client.MeshIntegration]
}

func (m MeshIntegrationClient) Create(_ context.Context, integration client.MeshIntegration) (*client.MeshIntegration, error) {
	integrationUuid := uuid.NewString()
	created := &client.MeshIntegration{
		Metadata: client.MeshIntegrationMetadata{
			Uuid:             new(integrationUuid),
			OwnedByWorkspace: integration.Metadata.OwnedByWorkspace,
		},
		Spec: integration.Spec,
		Status: &client.MeshIntegrationStatus{
			IsBuiltIn: false,
		},
	}
	backendSecretBehavior(true, created, nil)
	if created.Spec.Config.EntraId != nil {
		if created.Spec.Config.EntraId.IdpAlias == nil {
			created.Spec.Config.EntraId.IdpAlias = new(fmt.Sprintf("idp-integration-%s", integrationUuid))
		}
		if created.Spec.Config.EntraId.RedirectUrl == nil {
			created.Spec.Config.EntraId.RedirectUrl = new(fmt.Sprintf("https://meshstack.example.com/oauth/redirect/%s", *created.Spec.Config.EntraId.IdpAlias))
		}
	}
	m.Store.Set(integrationUuid, created)
	return created, nil
}

func (m MeshIntegrationClient) Read(_ context.Context, uuid string) (*client.MeshIntegration, error) {
	if integration, ok := m.Store.Get(uuid); ok {
		return integration, nil
	}
	return nil, nil
}

func (m MeshIntegrationClient) Update(_ context.Context, integration client.MeshIntegration) (*client.MeshIntegration, error) {
	if existing, ok := m.Store.Get(*integration.Metadata.Uuid); ok {
		backendSecretBehavior(false, &integration, existing)
		if integration.Spec.Config.EntraId != nil && existing.Spec.Config.EntraId != nil {
			// The alias is immutable: the backend ignores whatever the request carries.
			integration.Spec.Config.EntraId.IdpAlias = existing.Spec.Config.EntraId.IdpAlias
			if integration.Spec.Config.EntraId.RedirectUrl == nil {
				integration.Spec.Config.EntraId.RedirectUrl = existing.Spec.Config.EntraId.RedirectUrl
			}
		}
		existing.Spec = integration.Spec
		return existing, nil
	}
	return nil, fmt.Errorf("integration not found: %s", *integration.Metadata.Uuid)
}

func (m MeshIntegrationClient) Delete(_ context.Context, uuid string) error {
	m.Store.Delete(uuid)
	return nil
}

func (m MeshIntegrationClient) List(_ context.Context) ([]client.MeshIntegration, error) {
	var result []client.MeshIntegration
	for _, integration := range m.Store.Values() {
		result = append(result, *integration)
	}
	return result, nil
}
