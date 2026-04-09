package provider

import (
	"github.com/meshcloud/terraform-provider-meshstack/client"
)

type platformTypeModel struct {
	*client.MeshPlatformType
	Ref platformTypeRef `tfsdk:"ref"`
}

type platformTypeRef struct {
	Kind string `tfsdk:"kind"`
	Name string `tfsdk:"name"`
}

func newPlatformTypeModel(meshPlatformType *client.MeshPlatformType) platformTypeModel {
	return platformTypeModel{
		MeshPlatformType: meshPlatformType,
		Ref: platformTypeRef{
			Kind: client.MeshObjectKind.PlatformType,
			Name: meshPlatformType.Metadata.Name,
		},
	}
}
