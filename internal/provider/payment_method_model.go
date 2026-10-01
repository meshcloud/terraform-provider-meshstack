package provider

import (
	"github.com/meshcloud/meshstack-cli/client"
)

type paymentMethodModel struct {
	*client.MeshPaymentMethod
	Ref client.NamedRef `tfsdk:"ref"`
}

func newPaymentMethodModel(paymentMethod *client.MeshPaymentMethod) paymentMethodModel {
	return paymentMethodModel{
		MeshPaymentMethod: paymentMethod,
		Ref: client.NamedRef{
			Kind: client.MeshObjectKind.PaymentMethod,
			Name: paymentMethod.Metadata.Name,
		},
	}
}
