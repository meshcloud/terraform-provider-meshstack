package clientmock

import (
	"context"
	"fmt"
	"time"

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
			Name:             apiKey.Metadata.Name,
			OwnedByWorkspace: apiKey.Metadata.OwnedByWorkspace,
			CreatedOn:        time.Now().UTC().Format(time.RFC3339),
		},
		Spec: apiKey.Spec,
	}

	m.Store.Set(apiKeyUuid, stored)

	created := *stored
	created.Token = &token
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

	existing.Metadata.Name = apiKey.Metadata.Name
	existing.Metadata.OwnedByWorkspace = apiKey.Metadata.OwnedByWorkspace
	existing.Spec = apiKey.Spec

	return existing, nil
}

func (m *MeshApiKeyClient) Delete(_ context.Context, uuid string) error {
	m.DeleteCalls++
	m.Store.Delete(uuid)
	return nil
}
