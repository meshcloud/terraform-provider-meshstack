package provider

import (
	"github.com/meshcloud/meshstack-cli/client"
)

type locationModel struct {
	*client.MeshLocation
	Ref client.NamedRef `tfsdk:"ref"`
}

func newLocationModel(location *client.MeshLocation) locationModel {
	return locationModel{
		MeshLocation: location,
		Ref: client.NamedRef{
			Kind: client.MeshObjectKind.Location,
			Name: location.Metadata.Name,
		},
	}
}
