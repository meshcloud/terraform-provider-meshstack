package provider

import (
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/meshcloud/meshstack-cli/client"
)

type projectModel struct {
	Ref      projectRef                 `tfsdk:"ref"`
	Metadata client.MeshProjectMetadata `tfsdk:"metadata"`
	Spec     projectSpecModel           `tfsdk:"spec"`
}

type projectSpecModel struct {
	DisplayName                       string              `tfsdk:"display_name"`
	Tags                              map[string][]string `tfsdk:"tags"`
	PaymentMethodIdentifier           *string             `tfsdk:"payment_method_identifier"`
	PaymentMethodRef                  *client.NamedRef    `tfsdk:"payment_method_ref"`
	SubstitutePaymentMethodIdentifier *string             `tfsdk:"substitute_payment_method_identifier"`
	SubstitutePaymentMethodRef        *client.NamedRef    `tfsdk:"substitute_payment_method_ref"`
}

func newProjectModel(project client.MeshProject) projectModel {
	return projectModel{
		Ref:      newProjectRef(project.Metadata.OwnedByWorkspace, project.Metadata.Name),
		Metadata: project.Metadata,
		Spec: projectSpecModel{
			DisplayName:                       project.Spec.DisplayName,
			Tags:                              project.Spec.Tags,
			PaymentMethodIdentifier:           project.Spec.PaymentMethodIdentifier,
			PaymentMethodRef:                  paymentMethodRef(project.Spec.PaymentMethodIdentifier),
			SubstitutePaymentMethodIdentifier: project.Spec.SubstitutePaymentMethodIdentifier,
			SubstitutePaymentMethodRef:        paymentMethodRef(project.Spec.SubstitutePaymentMethodIdentifier),
		},
	}
}

func (m projectModel) meshProjectCreate() client.MeshProjectCreate {
	tags := m.Spec.Tags
	if tags == nil {
		tags = map[string][]string{}
	}
	return client.MeshProjectCreate{
		Metadata: client.MeshProjectCreateMetadata{
			Name:             m.Metadata.Name,
			OwnedByWorkspace: m.Metadata.OwnedByWorkspace,
		},
		Spec: client.MeshProjectSpec{
			DisplayName:                       m.Spec.DisplayName,
			Tags:                              tags,
			PaymentMethodIdentifier:           paymentMethodIdentifier(m.Spec.PaymentMethodIdentifier, m.Spec.PaymentMethodRef),
			SubstitutePaymentMethodIdentifier: paymentMethodIdentifier(m.Spec.SubstitutePaymentMethodIdentifier, m.Spec.SubstitutePaymentMethodRef),
		},
	}
}

func paymentMethodRef(identifier *string) *client.NamedRef {
	if identifier == nil {
		return nil
	}
	return &client.NamedRef{Kind: client.MeshObjectKind.PaymentMethod, Name: *identifier}
}

func paymentMethodIdentifier(identifier *string, ref *client.NamedRef) *string {
	if identifier != nil || ref == nil {
		return identifier
	}
	return &ref.Name
}

func projectPaymentMethodRefOutput() schema.SingleNestedAttribute {
	return meshRefByName(meshRefOptions{Kind: client.MeshObjectKind.PaymentMethod, Description: "Reference to the payment method of the project.", Output: true})
}

func projectSubstitutePaymentMethodRefOutput() schema.SingleNestedAttribute {
	return meshRefByName(meshRefOptions{Kind: client.MeshObjectKind.PaymentMethod, Description: "Reference to the payment method that meshStack makes the payment method of the project when `payment_method_ref` expires.", Output: true})
}
