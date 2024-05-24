package provider

type MeshProjectUserBinding struct {
	ApiVersion string                         `json:"apiVersion" tfsdk:"api_version"`
	Kind       string                         `json:"kind" tfsdk:"kind"`
	Metadata   MeshProjectUserBindingMetadata `json:"metadata" tfsdk:"metadata"`
	RoleRef    MeshProjectRoleRef             `json:"roleRef" tfsdk:"role_ref"`
	TargetRef  MeshProjectTargetRef           `json:"targetRef" tfsdk:"target_ref"`
	Subject    MeshSubject                    `json:"subject" tfsdk:"subject"`
}

type MeshProjectUserBindingMetadata struct {
	Name string `json:"name" tfsdk:"name"`
}

type MeshProjectRoleRef struct {
	Name string `json:"name" tfsdk:"name"`
}

type MeshProjectTargetRef struct {
	Name             string `json:"name" tfsdk:"name"`
	OwnedByWorkspace string `json:"ownedByWorkspace" tfsdk:"owned_by_workspace"`
}

type MeshSubject struct {
	Name string `json:"name" tfsdk:"name"`
}
