package clientmock

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	"github.com/meshcloud/terraform-provider-meshstack/client"
)

type MeshApiKeyClient struct {
	Store       *Store[client.MeshApiKey]
	CreateCalls int
	UpdateCalls int
	DeleteCalls int
}

func (m *MeshApiKeyClient) Create(_ context.Context, apiKey *client.MeshApiKeyCreate) (*client.MeshApiKey, error) {
	m.CreateCalls++

	apiKeyUuid := uuid.NewString()
	token := "token-" + uuid.NewString()

	stored := &client.MeshApiKey{
		Metadata: client.MeshApiKeyMetadata{
			Uuid:             &apiKeyUuid,
			OwnedByWorkspace: apiKey.Metadata.OwnedByWorkspace,
		},
		Spec: apiKey.Spec,
	}

	m.Store.Set(apiKeyUuid, stored)

	created := *stored
	created.Status = &client.MeshApiKeyStatus{Token: &token}
	return &created, nil
}

func (m *MeshApiKeyClient) Read(_ context.Context, uuid string) (*client.MeshApiKey, error) {
	if apiKey, ok := m.Store.Get(uuid); ok {
		return apiKey, nil
	}
	return nil, nil
}

func (m *MeshApiKeyClient) Update(_ context.Context, uuid string, apiKey *client.MeshApiKeyCreate) (*client.MeshApiKey, error) {
	m.UpdateCalls++

	existing, ok := m.Store.Get(uuid)
	if !ok {
		return nil, fmt.Errorf("api key not found: %s", uuid)
	}

	// Simulate secret rotation when expiresAt changes
	var rotatedToken *client.MeshApiKeyStatus
	if apiKey.Spec.ExpiresAt != nil && existing.Spec.ExpiresAt != nil && *apiKey.Spec.ExpiresAt != *existing.Spec.ExpiresAt {
		newToken := "rotated-token-" + uuid
		rotatedToken = &client.MeshApiKeyStatus{Token: &newToken}
	}

	existing.Spec = apiKey.Spec

	result := *existing
	result.Status = rotatedToken
	return &result, nil
}

func (m *MeshApiKeyClient) Delete(_ context.Context, uuid string) error {
	m.DeleteCalls++
	m.Store.Delete(uuid)
	return nil
}
