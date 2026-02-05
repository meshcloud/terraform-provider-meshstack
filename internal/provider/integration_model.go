package provider

import (
	"github.com/hashicorp/terraform-plugin-framework/diag"

	"github.com/meshcloud/terraform-provider-meshstack/client"
	"github.com/meshcloud/terraform-provider-meshstack/internal/types/generic"
)

type integration struct {
	Metadata client.MeshIntegrationMetadataAdapter[generic.Value[string]] `tfsdk:"metadata"`
	Spec     client.MeshIntegrationSpec                                   `tfsdk:"spec"`
	Status   generic.Value[client.MeshIntegrationStatus]                  `tfsdk:"status"`
}

func (model integration) ToClientDto(diags *diag.Diagnostics) client.MeshIntegration {
	return client.MeshIntegration{
		Metadata: client.MeshIntegrationMetadata{
			Uuid:             model.Metadata.Uuid.GetPtr(diags),
			OwnedByWorkspace: model.Metadata.OwnedByWorkspace,
		},
		Spec: client.MeshIntegrationSpec{
			DisplayName: model.Spec.DisplayName,
			Config:      model.Spec.Config,
		},
	}
}

func (model *integration) SetFromClientDto(dto *client.MeshIntegration, diags *diag.Diagnostics) {
	model.Metadata.Uuid.SetRequired(dto.Metadata.Uuid, diags)
	model.Metadata.OwnedByWorkspace = dto.Metadata.OwnedByWorkspace
	model.Spec.DisplayName = dto.Spec.DisplayName
	model.Spec.Config = dto.Spec.Config
	model.Status.SetRequired(dto.Status, diags)
}
