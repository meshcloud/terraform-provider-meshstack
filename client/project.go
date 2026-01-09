package client

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

type MeshProjectClient struct {
	meshObjectClient[MeshProject]
}

func newProjectClient(c *httpClient) MeshProjectClient {
	return MeshProjectClient{newMeshObjectClient[MeshProject](c, "v2")}
}

func (c MeshProjectClient) projectId(workspace string, name string) string {
	return workspace + "." + name
}

func (c MeshProjectClient) Read(workspace string, name string) (*MeshProject, error) {
	return c.get(c.projectId(workspace, name))
}

func (c MeshProjectClient) List(workspaceIdentifier string, paymentMethodIdentifier *string) ([]MeshProject, error) {
	options := []doRequestOption{
		withUrlQuery("workspaceIdentifier", workspaceIdentifier),
	}
	if paymentMethodIdentifier != nil {
		options = append(options, withUrlQuery("paymentIdentifier", *paymentMethodIdentifier))
	}
	return c.list(options...)
}

func (c MeshProjectClient) Create(project *MeshProjectCreate) (*MeshProject, error) {
	return c.post(project)
}

func (c MeshProjectClient) Update(project *MeshProjectCreate) (*MeshProject, error) {
	return c.put(c.projectId(project.Metadata.OwnedByWorkspace, project.Metadata.Name), project)
}

func (c MeshProjectClient) Delete(workspace string, name string) error {
	return c.delete(c.projectId(workspace, name))
}
