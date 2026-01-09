package client

type MeshWorkspace struct {
	ApiVersion string                `json:"apiVersion" tfsdk:"api_version"`
	Kind       string                `json:"kind" tfsdk:"kind"`
	Metadata   MeshWorkspaceMetadata `json:"metadata" tfsdk:"metadata"`
	Spec       MeshWorkspaceSpec     `json:"spec" tfsdk:"spec"`
}

type MeshWorkspaceMetadata struct {
	Name      string              `json:"name" tfsdk:"name"`
	CreatedOn string              `json:"createdOn" tfsdk:"created_on"`
	DeletedOn *string             `json:"deletedOn" tfsdk:"deleted_on"`
	Tags      map[string][]string `json:"tags" tfsdk:"tags"`
}

type MeshWorkspaceSpec struct {
	DisplayName                  string `json:"displayName" tfsdk:"display_name"`
	PlatformBuilderAccessEnabled *bool  `json:"platformBuilderAccessEnabled,omitempty" tfsdk:"platform_builder_access_enabled"`
}

type MeshWorkspaceCreate struct {
	ApiVersion string                      `json:"apiVersion" tfsdk:"api_version"`
	Metadata   MeshWorkspaceCreateMetadata `json:"metadata" tfsdk:"metadata"`
	Spec       MeshWorkspaceSpec           `json:"spec" tfsdk:"spec"`
}
type MeshWorkspaceCreateMetadata struct {
	Name string              `json:"name" tfsdk:"name"`
	Tags map[string][]string `json:"tags" tfsdk:"tags"`
}

type MeshWorkspaceClient struct {
	meshObjectClient[MeshWorkspace]
}

func newWorkspaceClient(c *httpClient) MeshWorkspaceClient {
	return MeshWorkspaceClient{newMeshObjectClient[MeshWorkspace](c, "v2")}
}

func (c MeshWorkspaceClient) Read(name string) (*MeshWorkspace, error) {
	return c.get(name)
}

func (c MeshWorkspaceClient) Create(workspace *MeshWorkspaceCreate) (*MeshWorkspace, error) {
	return c.post(workspace)
}

func (c MeshWorkspaceClient) Update(name string, workspace *MeshWorkspaceCreate) (*MeshWorkspace, error) {
	return c.put(name, workspace)
}

func (c MeshWorkspaceClient) Delete(name string) error {
	return c.delete(name)
}
