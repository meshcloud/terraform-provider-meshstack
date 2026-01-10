package client

type MeshWorkspaceBinding struct {
	ApiVersion string                       `json:"apiVersion" tfsdk:"api_version"`
	Kind       string                       `json:"kind" tfsdk:"kind"`
	Metadata   MeshWorkspaceBindingMetadata `json:"metadata" tfsdk:"metadata"`
	RoleRef    MeshWorkspaceRoleRef         `json:"roleRef" tfsdk:"role_ref"`
	TargetRef  MeshWorkspaceTargetRef       `json:"targetRef" tfsdk:"target_ref"`
	Subject    MeshWorkspaceSubject         `json:"subject" tfsdk:"subject"`
}

type MeshWorkspaceBindingMetadata struct {
	Name string `json:"name" tfsdk:"name"`
}

type MeshWorkspaceRoleRef struct {
	Name string `json:"name" tfsdk:"name"`
}

type MeshWorkspaceTargetRef struct {
	Name string `json:"name" tfsdk:"name"`
}

type MeshWorkspaceSubject struct {
	Name string `json:"name" tfsdk:"name"`
}
