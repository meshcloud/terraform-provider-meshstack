package client

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
)

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

func (c *MeshStackProviderClient) urlForPojectUserBinding(name string) *url.URL {
	return c.endpoints.ProjectUserBindings.JoinPath(name)
}

func (c *MeshStackProviderClient) ReadProjectUserBinding(name string) (*MeshProjectUserBinding, error) {
	targetUrl := c.urlForPojectUserBinding(name)
	req, err := http.NewRequest("GET", targetUrl.String(), nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", CONTENT_TYPE_PROJECT_USER_BINDING)

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

	if res.StatusCode != 200 {
		return nil, fmt.Errorf("unexpected status code: %d, %s", res.StatusCode, data)
	}

	var binding MeshProjectUserBinding
	err = json.Unmarshal(data, &binding)
	if err != nil {
		return nil, err
	}

	return &binding, nil
}

func (c *MeshStackProviderClient) CreateProjectUserBinding(binding *MeshProjectUserBinding) (*MeshProjectUserBinding, error) {
	payload, err := json.Marshal(binding)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest("POST", c.endpoints.ProjectUserBindings.String(), bytes.NewBuffer(payload))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", CONTENT_TYPE_PROJECT_USER_BINDING)
	req.Header.Set("Accept", CONTENT_TYPE_PROJECT_USER_BINDING)

	res, err := c.doAuthenticatedRequest(req)
	if err != nil {
		return nil, err
	}

	defer res.Body.Close()

	data, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}

	if res.StatusCode != 200 {
		return nil, fmt.Errorf("unexpected status code: %d, %s", res.StatusCode, data)
	}

	var createdBinding MeshProjectUserBinding
	err = json.Unmarshal(data, &createdBinding)
	if err != nil {
		return nil, err
	}

	return &createdBinding, nil
}

func (c *MeshStackProviderClient) DeleteProjecUserBinding(name string) error {
	targetUrl := c.urlForPojectUserBinding(name)
	return c.deleteMeshObject(*targetUrl, 204)
}
