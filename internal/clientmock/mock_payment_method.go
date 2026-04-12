package clientmock

import (
	"context"
	"fmt"
	"time"

	"github.com/meshcloud/terraform-provider-meshstack/client"
)

type MeshPaymentMethodClient struct {
	Store Store[client.MeshPaymentMethod]
}

func (m MeshPaymentMethodClient) Read(_ context.Context, workspace string, identifier string) (*client.MeshPaymentMethod, error) {
	return m.Store[identifier], nil
}

func (m MeshPaymentMethodClient) Create(_ context.Context, paymentMethod *client.MeshPaymentMethodCreate) (*client.MeshPaymentMethod, error) {
	created := &client.MeshPaymentMethod{
		Metadata: client.MeshPaymentMethodMetadata{
			Name:             paymentMethod.Metadata.Name,
			OwnedByWorkspace: paymentMethod.Metadata.OwnedByWorkspace,
			CreatedOn:        time.Now().UTC().Format(time.RFC3339),
		},
		Spec: paymentMethod.Spec,
	}

	if created.Spec.Tags == nil {
		created.Spec.Tags = map[string][]string{}
	}

	m.Store[paymentMethod.Metadata.Name] = created
	return created, nil
}

func (m MeshPaymentMethodClient) Update(_ context.Context, identifier string, paymentMethod *client.MeshPaymentMethodCreate) (*client.MeshPaymentMethod, error) {
	existing := m.Store[identifier]
	if existing == nil {
		return nil, fmt.Errorf("payment method not found: %s", identifier)
	}

	existing.Spec = paymentMethod.Spec
	if existing.Spec.Tags == nil {
		existing.Spec.Tags = map[string][]string{}
	}

	return existing, nil
}

func (m MeshPaymentMethodClient) Delete(_ context.Context, identifier string) error {
	delete(m.Store, identifier)
	return nil
}
