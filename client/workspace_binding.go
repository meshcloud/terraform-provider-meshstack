package client

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
)

type MeshWorkspaceBinding struct {
	ApiVersion string                     `json:"apiVersion" tfsdk:"api_version"`
	Kind       string                     `json:"kind" tfsdk:"kind"`
	Metadata   MeshWorkspaceBindingMetadata `json:"metadata" tfsdk:"metadata"`
	RoleRef    MeshWorkspaceRoleRef         `json:"roleRef" tfsdk:"role_ref"`
	TargetRef  MeshWorkspaceTargetRef       `json:"targetRef" tfsdk:"target_ref"`
	Subject    MeshWorkspaceSubject                `json:"subject" tfsdk:"subject"`
}

type MeshWorkspaceBindingMetadata struct {
	Name string `json:"name" tfsdk:"name"`
}

type MeshWorkspaceRoleRef struct {
	Name string `json:"name" tfsdk:"name"`
}

type MeshWorkspaceTargetRef struct {
	Name             string `json:"name" tfsdk:"name"`
}

type MeshWorkspaceSubject struct {
	Name string `json:"name" tfsdk:"name"`
}

func (c *MeshStackProviderClient) readWorkspaceBinding(name string, contentType string) (*MeshWorkspaceBinding, error) {
	var targetUrl *url.URL
	switch contentType {
	case CONTENT_TYPE_WORKSPACE_USER_BINDING:
		targetUrl = c.urlForWorkspaceUserBinding(name)

	case CONTENT_TYPE_WORKSPACE_GROUP_BINDING:
		targetUrl = c.urlForWorkspaceGroupBinding(name)

	default:
		return nil, fmt.Errorf("Unexpected content type: %s", contentType)
	}

	req, err := http.NewRequest("GET", targetUrl.String(), nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", contentType)

	res, err := c.doAuthenticatedRequest(req)
	if err != nil {
		return nil, err
	}

	defer res.Body.Close()

	data, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}

	if res.StatusCode == 404 {
		return nil, nil
	}

	if !isSuccessHTTPStatus(res) {
		return nil, fmt.Errorf("unexpected status code: %d, %s", res.StatusCode, data)
	}

	var binding MeshWorkspaceBinding
	err = json.Unmarshal(data, &binding)
	if err != nil {
		return nil, err
	}

	return &binding, nil
}

func (c *MeshStackProviderClient) createWorkspaceBinding(binding *MeshWorkspaceBinding, contentType string) (*MeshWorkspaceBinding, error) {
	var targetUrl *url.URL
	switch contentType {
	case CONTENT_TYPE_WORKSPACE_USER_BINDING:
		targetUrl = c.endpoints.WorkspaceUserBindings

	case CONTENT_TYPE_WORKSPACE_GROUP_BINDING:
		targetUrl = c.endpoints.WorkspaceGroupBindings

	default:
		return nil, fmt.Errorf("Unexpected content type: %s", contentType)
	}

	payload, err := json.Marshal(binding)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest("POST", targetUrl.String(), bytes.NewBuffer(payload))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", contentType)
	req.Header.Set("Accept", contentType)

	res, err := c.doAuthenticatedRequest(req)
	if err != nil {
		return nil, err
	}

	defer res.Body.Close()

	data, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}

	if !isSuccessHTTPStatus(res) {
		return nil, fmt.Errorf("unexpected status code: %d, %s", res.StatusCode, data)
	}

	var createdBinding MeshWorkspaceBinding
	err = json.Unmarshal(data, &createdBinding)
	if err != nil {
		return nil, err
	}

	return &createdBinding, nil
}
