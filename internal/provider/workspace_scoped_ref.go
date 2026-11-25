package provider

import "github.com/meshcloud/meshstack-cli/client"

type workspaceScopedRef struct {
	Kind             string `tfsdk:"kind"`
	OwnedByWorkspace string `tfsdk:"owned_by_workspace"`
	Name             string `tfsdk:"name"`
}

func newProjectRef(workspace, name string) workspaceScopedRef {
	return workspaceScopedRef{Kind: client.MeshObjectKind.Project, OwnedByWorkspace: workspace, Name: name}
}

func newWorkspaceUserGroupRef(workspace, name string) workspaceScopedRef {
	return workspaceScopedRef{Kind: client.MeshObjectKind.WorkspaceUserGroup, OwnedByWorkspace: workspace, Name: name}
}
