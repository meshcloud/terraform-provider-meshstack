package clientmock

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"

	"github.com/meshcloud/terraform-provider-meshstack/client"
	"github.com/meshcloud/terraform-provider-meshstack/client/types/ptr"
)

type MeshIntegrationClient struct {
	Store Store[client.MeshIntegration]
}

func (m MeshIntegrationClient) Create(_ context.Context, integration client.MeshIntegration) (*client.MeshIntegration, error) {
	integrationUuid := acctest.RandString(32)
	created := &client.MeshIntegration{
		ApiVersion: integration.ApiVersion,
		Kind:       integration.Kind,
		Metadata: client.MeshIntegrationMetadata{
			Uuid:             ptr.To(integrationUuid),
			OwnedByWorkspace: integration.Metadata.OwnedByWorkspace,
		},
		Spec: integration.Spec,
		Status: &client.MeshIntegrationStatus{
			IsBuiltIn: false,
			WorkloadIdentityFederation: &client.MeshWorkloadIdentityFederation{
				Issuer:  "https://meshstack.example.com",
				Subject: "integration:" + integrationUuid,
				Gcp: &client.MeshWifProvider{
					Audience: "gcp-audience",
				},
				Aws: &client.MeshAwsWifProvider{
					Audience:   "aws-audience",
					Thumbprint: "abc123",
				},
				Azure: &client.MeshWifProvider{
					Audience: "azure-audience",
				},
			},
		},
	}
	m.Store[integrationUuid] = created
	return created, nil
}

func (m MeshIntegrationClient) Read(_ context.Context, uuid string) (*client.MeshIntegration, error) {
	if integration, ok := m.Store[uuid]; ok {
		return integration, nil
	}
	return nil, nil
}

func (m MeshIntegrationClient) Update(_ context.Context, integration client.MeshIntegration) (*client.MeshIntegration, error) {
	if existing, ok := m.Store[*integration.Metadata.Uuid]; ok {
		existing.Spec = integration.Spec
		return existing, nil
	}
	return nil, fmt.Errorf("integration not found: %s", *integration.Metadata.Uuid)
}

func (m MeshIntegrationClient) Delete(_ context.Context, uuid string) error {
	delete(m.Store, uuid)
	return nil
}

func (m MeshIntegrationClient) List(_ context.Context) ([]client.MeshIntegration, error) {
	var result []client.MeshIntegration
	for _, integration := range m.Store {
		result = append(result, *integration)
	}
	return result, nil
}
