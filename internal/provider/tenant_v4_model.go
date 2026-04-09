package provider

import (
	"github.com/meshcloud/terraform-provider-meshstack/client"
)

type tenantV4Model struct {
	*client.MeshTenantV4
	Ref tenantV4Ref `tfsdk:"ref"`
}

type tenantV4Ref struct {
	Kind string `tfsdk:"kind"`
	Uuid string `tfsdk:"uuid"`
}

func newTenantV4Model(tenant *client.MeshTenantV4) tenantV4Model {
	return tenantV4Model{
		MeshTenantV4: tenant,
		Ref: tenantV4Ref{
			Kind: client.MeshObjectKind.Tenant,
			Uuid: tenant.Metadata.Uuid,
		},
	}
}
