package client

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
)

const CONTENT_TYPE_WORKSPACE = "application/vnd.meshcloud.api.meshworkspace.v1.hal+json"

type MeshWorkspace struct {
	ApiVersion string                `json:"apiVersion" tfsdk:"api_version"`
	Kind       string                `json:"kind" tfsdk:"kind"`
	Metadata   MeshWorkspaceMetadata `json:"metadata" tfsdk:"metadata"`
	Spec       MeshWorkspaceSpec     `json:"spec" tfsdk:"spec"`
}

type MeshWorkspaceMetadata struct {
	Name      string  `json:"name" tfsdk:"name"`
	CreatedOn string  `json:"createdOn" tfsdk:"created_on"`
	DeletedOn *string `json:"deletedOn" tfsdk:"deleted_on"`
}

type MeshWorkspaceSpec struct {
	DisplayName string              `json:"displayName" tfsdk:"display_name"`
	Tags        map[string][]string `json:"tags" tfsdk:"tags"`
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

	res, err := c.doAuthenticatedRequest(req)
	if err != nil {
		return nil, err
	}

	defer res.Body.Close()

	if res.StatusCode == http.StatusNotFound {
		return nil, nil // Not found is not an error
	}

	data, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}

	if !isSuccessHTTPStatus(res) {
		return nil, fmt.Errorf("unexpected status code: %d, %s", res.StatusCode, data)
	}

	var workspace MeshWorkspace
	err = json.Unmarshal(data, &workspace)
	if err != nil {
		return nil, err
	}
	return &workspace, nil
}
