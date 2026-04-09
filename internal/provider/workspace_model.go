package provider

import (
	"github.com/meshcloud/terraform-provider-meshstack/client"
)

type workspaceModel struct {
	*client.MeshWorkspace
	Ref workspaceRef `tfsdk:"ref"`
}

type workspaceRef struct {
	Kind       string `tfsdk:"kind"`
	Identifier string `tfsdk:"identifier"`
}

func newWorkspaceModel(workspace *client.MeshWorkspace) workspaceModel {
	return workspaceModel{
		MeshWorkspace: workspace,
		Ref: workspaceRef{
			Kind:       client.MeshObjectKind.Workspace,
			Identifier: workspace.Metadata.Name,
		},
	}
}
