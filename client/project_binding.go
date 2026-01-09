package client

import (
	"fmt"
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

	return unmarshalBodyIfPresent[MeshProjectBinding](c.doAuthenticatedRequest("GET", targetUrl,
		withAccept(contentType),
	))
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

	return unmarshalBody[MeshProjectBinding](c.doAuthenticatedRequest("POST", targetUrl,
		withPayload(binding, contentType),
	))
}
