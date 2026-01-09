package client

import (
	"bytes"
	"encoding/json"
	"net/http"
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
	targetUrl := c.urlForWorkspace(name)
	req, err := http.NewRequest("GET", targetUrl.String(), nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", CONTENT_TYPE_WORKSPACE)

	return unmarshalBodyIfPresent[MeshWorkspace](c.doAuthenticatedRequest(req))
}

func (c *MeshStackProviderClient) CreateWorkspace(workspace *MeshWorkspaceCreate) (*MeshWorkspace, error) {
	payload, err := json.Marshal(workspace)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest("POST", c.endpoints.Workspaces.String(), bytes.NewBuffer(payload))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", CONTENT_TYPE_WORKSPACE)
	req.Header.Set("Accept", CONTENT_TYPE_WORKSPACE)

	return unmarshalBody[MeshWorkspace](c.doAuthenticatedRequest(req))
}

func (c *MeshStackProviderClient) UpdateWorkspace(name string, workspace *MeshWorkspaceCreate) (*MeshWorkspace, error) {
	targetUrl := c.urlForWorkspace(name)

	payload, err := json.Marshal(workspace)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest("PUT", targetUrl.String(), bytes.NewBuffer(payload))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", CONTENT_TYPE_WORKSPACE)
	req.Header.Set("Accept", CONTENT_TYPE_WORKSPACE)

	return unmarshalBody[MeshWorkspace](c.doAuthenticatedRequest(req))
}

func (c *MeshStackProviderClient) DeleteWorkspace(name string) error {
	targetUrl := c.urlForWorkspace(name)
	return c.deleteMeshObject(*targetUrl, 204)
}
