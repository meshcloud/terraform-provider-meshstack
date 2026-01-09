package client

import (
	"net/url"
)

const CONTENT_TYPE_WORKSPACE = "application/vnd.meshcloud.api.meshworkspace.v2.hal+json"

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

func (c *MeshStackProviderClient) urlForWorkspace(name string) *url.URL {
	return c.endpoints.Workspaces.JoinPath(name)
}

func (c *MeshStackProviderClient) ReadWorkspace(name string) (*MeshWorkspace, error) {
	return unmarshalBodyIfPresent[MeshWorkspace](c.doAuthenticatedRequest("GET", c.urlForWorkspace(name),
		withAccept(CONTENT_TYPE_WORKSPACE),
	))
}

func (c *MeshStackProviderClient) CreateWorkspace(workspace *MeshWorkspaceCreate) (*MeshWorkspace, error) {
	return unmarshalBody[MeshWorkspace](c.doAuthenticatedRequest("POST", c.endpoints.Workspaces,
		withPayload(workspace, CONTENT_TYPE_WORKSPACE),
	))
}

func (c *MeshStackProviderClient) UpdateWorkspace(name string, workspace *MeshWorkspaceCreate) (*MeshWorkspace, error) {
	return unmarshalBody[MeshWorkspace](c.doAuthenticatedRequest("PUT", c.urlForWorkspace(name),
		withPayload(workspace, CONTENT_TYPE_WORKSPACE),
	))
}

func (c *MeshStackProviderClient) DeleteWorkspace(name string) error {
	_, err := c.doAuthenticatedRequest("DELETE", c.urlForWorkspace(name),
		withAccept(CONTENT_TYPE_WORKSPACE),
	)
	return err
}
