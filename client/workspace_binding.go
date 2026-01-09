package client

import (
	"fmt"
	"net/url"
)

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

func (c *MeshStackProviderClient) readWorkspaceBinding(name string, contentType string) (*MeshWorkspaceBinding, error) {
	var targetUrl *url.URL
	switch contentType {
	case CONTENT_TYPE_WORKSPACE_USER_BINDING:
		targetUrl = c.urlForWorkspaceUserBinding(name)

	case CONTENT_TYPE_WORKSPACE_GROUP_BINDING:
		targetUrl = c.urlForWorkspaceGroupBinding(name)

	default:
		return nil, fmt.Errorf("unexpected content type '%s'", contentType)
	}

	return unmarshalBodyIfPresent[MeshWorkspaceBinding](c.doAuthenticatedRequest("GET", targetUrl,
		withAccept(contentType),
	))
}

func (c *MeshStackProviderClient) createWorkspaceBinding(binding *MeshWorkspaceBinding, contentType string) (*MeshWorkspaceBinding, error) {
	var targetUrl *url.URL
	switch contentType {
	case CONTENT_TYPE_WORKSPACE_USER_BINDING:
		targetUrl = c.endpoints.WorkspaceUserBindings
	case CONTENT_TYPE_WORKSPACE_GROUP_BINDING:
		targetUrl = c.endpoints.WorkspaceGroupBindings
	default:
		return nil, fmt.Errorf("unexpected content type '%s'", contentType)
	}

	return unmarshalBody[MeshWorkspaceBinding](c.doAuthenticatedRequest("POST", targetUrl,
		withPayload(binding, contentType),
	))
}
