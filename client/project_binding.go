package client

type MeshProjectBinding struct {
	ApiVersion string                     `json:"apiVersion" tfsdk:"api_version"`
	Kind       string                     `json:"kind" tfsdk:"kind"`
	Metadata   MeshProjectBindingMetadata `json:"metadata" tfsdk:"metadata"`
	RoleRef    MeshProjectRoleRef         `json:"roleRef" tfsdk:"role_ref"`
	TargetRef  MeshProjectTargetRef       `json:"targetRef" tfsdk:"target_ref"`
	Subject    MeshSubject                `json:"subject" tfsdk:"subject"`
}

type MeshProjectBindingMetadata struct {
	Name string `json:"name" tfsdk:"name"`
}

// Deprecated: Use MeshProjectRoleRefV2 if possible. The convention is to also provide the `kind`,
// so this struct should only be used for meshobjects that violate our API conventions.
type MeshProjectRoleRef struct {
	Name string `json:"name" tfsdk:"name"`
}

type MeshProjectRoleRefV2 struct {
	Name string `json:"name" tfsdk:"name"`
	Kind string `json:"kind" tfsdk:"kind"`
}

type MeshProjectTargetRef struct {
	Name             string `json:"name" tfsdk:"name"`
	OwnedByWorkspace string `json:"ownedByWorkspace" tfsdk:"owned_by_workspace"`
}

type MeshSubject struct {
	Name string `json:"name" tfsdk:"name"`
}
