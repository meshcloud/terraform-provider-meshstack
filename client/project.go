package client

import (
	"net/url"
)

const CONTENT_TYPE_PROJECT = "application/vnd.meshcloud.api.meshproject.v2.hal+json"

type MeshProject struct {
	ApiVersion string              `json:"apiVersion" tfsdk:"api_version"`
	Kind       string              `json:"kind" tfsdk:"kind"`
	Metadata   MeshProjectMetadata `json:"metadata" tfsdk:"metadata"`
	Spec       MeshProjectSpec     `json:"spec" tfsdk:"spec"`
}

type MeshProjectMetadata struct {
	Name             string  `json:"name" tfsdk:"name"`
	OwnedByWorkspace string  `json:"ownedByWorkspace" tfsdk:"owned_by_workspace"`
	CreatedOn        string  `json:"createdOn" tfsdk:"created_on"`
	DeletedOn        *string `json:"deletedOn" tfsdk:"deleted_on"`
}

type MeshProjectSpec struct {
	DisplayName                       string              `json:"displayName" tfsdk:"display_name"`
	Tags                              map[string][]string `json:"tags" tfsdk:"tags"`
	PaymentMethodIdentifier           *string             `json:"paymentMethodIdentifier" tfsdk:"payment_method_identifier"`
	SubstitutePaymentMethodIdentifier *string             `json:"substitutePaymentMethodIdentifier" tfsdk:"substitute_payment_method_identifier"`
}

type MeshProjectCreate struct {
	Metadata MeshProjectCreateMetadata `json:"metadata" tfsdk:"metadata"`
	Spec     MeshProjectSpec           `json:"spec" tfsdk:"spec"`
}

type MeshProjectCreateMetadata struct {
	Name             string `json:"name" tfsdk:"name"`
	OwnedByWorkspace string `json:"ownedByWorkspace" tfsdk:"owned_by_workspace"`
}

func (c *MeshStackProviderClient) urlForProject(workspace string, name string) *url.URL {
	identifier := workspace + "." + name
	return c.endpoints.Projects.JoinPath(identifier)
}

func (c *MeshStackProviderClient) ReadProject(workspace string, name string) (*MeshProject, error) {
	return unmarshalBodyIfPresent[MeshProject](c.doAuthenticatedRequest("GET", c.urlForProject(workspace, name),
		withAccept(CONTENT_TYPE_PROJECT),
	))
}

func (c *MeshStackProviderClient) ReadProjects(workspaceIdentifier string, paymentMethodIdentifier *string) ([]MeshProject, error) {
	options := []doRequestOption{
		withAccept(CONTENT_TYPE_PROJECT),
		withUrlQuery("workspaceIdentifier", workspaceIdentifier),
	}
	if paymentMethodIdentifier != nil {
		options = append(options, withUrlQuery("paymentIdentifier", *paymentMethodIdentifier))
	}
	return unmarshalBodyPages[MeshProject]("meshProjects", c.doPaginatedRequest(c.endpoints.Projects, options...))
}

func (c *MeshStackProviderClient) CreateProject(project *MeshProjectCreate) (*MeshProject, error) {
	return unmarshalBody[MeshProject](c.doAuthenticatedRequest("POST", c.endpoints.Projects,
		withPayload(project, CONTENT_TYPE_PROJECT),
	))
}

func (c *MeshStackProviderClient) UpdateProject(project *MeshProjectCreate) (*MeshProject, error) {
	return unmarshalBody[MeshProject](c.doAuthenticatedRequest("PUT", c.urlForProject(project.Metadata.OwnedByWorkspace, project.Metadata.Name),
		withPayload(project, CONTENT_TYPE_PROJECT),
	))
}

func (c *MeshStackProviderClient) DeleteProject(workspace string, name string) error {
	_, err := c.doAuthenticatedRequest("DELETE", c.urlForProject(workspace, name),
		withAccept(CONTENT_TYPE_PROJECT),
	)
	return err
}
