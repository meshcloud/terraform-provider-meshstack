package provider

import "github.com/meshcloud/meshstack-cli/client"

type projectRef struct {
	Kind             string `tfsdk:"kind"`
	OwnedByWorkspace string `tfsdk:"owned_by_workspace"`
	Name             string `tfsdk:"name"`
}

func newProjectRef(workspace, name string) projectRef {
	return projectRef{Kind: client.MeshObjectKind.Project, OwnedByWorkspace: workspace, Name: name}
}
