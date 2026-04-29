package clientmock

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	"github.com/meshcloud/terraform-provider-meshstack/client"
)

type MeshApiKeyClient struct {
	Store *Store[client.MeshApiKey]
}

func (m MeshApiKeyClient) Create(_ context.Context, apiKey *client.MeshApiKey) (*client.MeshApiKey, error) {
	apiKeyUuid := uuid.NewString()
	stored := &client.MeshApiKey{
		Metadata: client.MeshApiKeyMetadata{
			Uuid:             &apiKeyUuid,
			OwnedByWorkspace: apiKey.Metadata.OwnedByWorkspace,
		},
		Spec: apiKey.Spec,
	}

	m.Store.Set(apiKeyUuid, stored)

	created := *stored
	created.Status = &client.MeshApiKeyStatus{ClientId: apiKeyUuid, ClientSecret: new("secret-" + uuid.NewString())}
	return &created, nil
}

func (m MeshApiKeyClient) Read(_ context.Context, uuid string) (*client.MeshApiKey, error) {
	if apiKey, ok := m.Store.Get(uuid); ok {
		result := *apiKey
		result.Status = &client.MeshApiKeyStatus{ClientId: uuid}
		return &result, nil
	}
	return nil, nil
}

func (m MeshApiKeyClient) Update(_ context.Context, uuid string, apiKey *client.MeshApiKey) (*client.MeshApiKey, error) {
	existing, ok := m.Store.Get(uuid)
	if !ok {
		return nil, fmt.Errorf("api key not found: %s", uuid)
	}

	expiresAtChanged := (existing.Spec.ExpiresAt == nil) != (apiKey.Spec.ExpiresAt == nil) ||
		(existing.Spec.ExpiresAt != nil && apiKey.Spec.ExpiresAt != nil && *existing.Spec.ExpiresAt != *apiKey.Spec.ExpiresAt)
	existing.Spec = apiKey.Spec

	result := *existing
	result.Status = &client.MeshApiKeyStatus{ClientId: uuid}

	if expiresAtChanged {
		// Secret is rotated when expires_at changes.
		clientSecret := "rotated-secret-" + uuid
		result.Status.ClientSecret = &clientSecret
	}

	return &result, nil
}

func (m MeshApiKeyClient) Delete(_ context.Context, uuid string) error {
	m.Store.Delete(uuid)
	return nil
}
