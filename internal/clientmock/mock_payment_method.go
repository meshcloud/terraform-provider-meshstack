package clientmock

import (
	"context"
	"fmt"
	"time"

	"github.com/meshcloud/meshstack-cli/client"
)

type MeshPaymentMethodClient struct {
	client.MeshPaymentMethodClient

	Store *Store[client.MeshPaymentMethod]
}

func (m MeshPaymentMethodClient) Read(_ context.Context, identifier string) (*client.MeshPaymentMethod, error) {
	v, _ := m.Store.Get(identifier)
	return copyPaymentMethod(v), nil
}

func (m MeshPaymentMethodClient) Create(_ context.Context, paymentMethod client.MeshPaymentMethod) (client.MeshPaymentMethod, error) {
	created := &client.MeshPaymentMethod{
		Metadata: client.MeshPaymentMethodMetadata{
			Name:             paymentMethod.Metadata.Name,
			OwnedByWorkspace: paymentMethod.Metadata.OwnedByWorkspace,
			CreatedOn:        time.Now().UTC().Format(time.RFC3339),
		},
		Spec: paymentMethod.Spec,
	}

	created.Spec.Tags = copyTags(paymentMethod.Spec.Tags)
	m.Store.Set(paymentMethod.Metadata.Name, created)
	return *copyPaymentMethod(created), nil
}

func (m MeshPaymentMethodClient) Update(_ context.Context, paymentMethod client.MeshPaymentMethod) (client.MeshPaymentMethod, error) {
	identifier := paymentMethod.Metadata.Name
	existing, _ := m.Store.Get(identifier)
	if existing == nil {
		return client.MeshPaymentMethod{}, fmt.Errorf("payment method not found: %s", identifier)
	}

	existing.Spec = paymentMethod.Spec
	existing.Spec.Tags = copyTags(paymentMethod.Spec.Tags)
	return *copyPaymentMethod(existing), nil
}

func (m MeshPaymentMethodClient) Delete(_ context.Context, identifier string) error {
	m.Store.Delete(identifier)
	return nil
}

func copyPaymentMethod(paymentMethod *client.MeshPaymentMethod) *client.MeshPaymentMethod {
	if paymentMethod == nil {
		return nil
	}
	copied := *paymentMethod
	copied.Spec.Tags = copyTags(paymentMethod.Spec.Tags)
	return &copied
}
