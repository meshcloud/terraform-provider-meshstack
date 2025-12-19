package client

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
)

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

func (c *MeshStackProviderClient) readProjectBinding(name string, contentType string) (*MeshProjectBinding, error) {
	var targetUrl *url.URL
	switch contentType {
	case CONTENT_TYPE_PROJECT_USER_BINDING:
		targetUrl = c.urlForPojectUserBinding(name)

	case CONTENT_TYPE_PROJECT_GROUP_BINDING:
		targetUrl = c.urlForPojectGroupBinding(name)

	default:
		return nil, fmt.Errorf("unexpected content type '%s'", contentType)
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

	defer func() {
		_ = res.Body.Close()
	}()

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

	var binding MeshProjectBinding
	err = json.Unmarshal(data, &binding)
	if err != nil {
		return nil, err
	}

	return &binding, nil
}

func (c *MeshStackProviderClient) createProjectBinding(binding *MeshProjectBinding, contentType string) (*MeshProjectBinding, error) {
	var targetUrl *url.URL
	switch contentType {
	case CONTENT_TYPE_PROJECT_USER_BINDING:
		targetUrl = c.endpoints.ProjectUserBindings

	case CONTENT_TYPE_PROJECT_GROUP_BINDING:
		targetUrl = c.endpoints.ProjectGroupBindings

	default:
		return nil, fmt.Errorf("unexpected content type '%s'", contentType)
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

	defer func() {
		_ = res.Body.Close()
	}()

	data, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}

	if !isSuccessHTTPStatus(res) {
		return nil, fmt.Errorf("unexpected status code: %d, %s", res.StatusCode, data)
	}

	var createdBinding MeshProjectBinding
	err = json.Unmarshal(data, &createdBinding)
	if err != nil {
		return nil, err
	}

	return &createdBinding, nil
}
