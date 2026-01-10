package client

import (
	"github.com/meshcloud/terraform-provider-meshstack/client/internal"
)

type MeshPaymentMethod struct {
	ApiVersion string                    `json:"apiVersion" tfsdk:"api_version"`
	Kind       string                    `json:"kind" tfsdk:"kind"`
	Metadata   MeshPaymentMethodMetadata `json:"metadata" tfsdk:"metadata"`
	Spec       MeshPaymentMethodSpec     `json:"spec" tfsdk:"spec"`
}

type MeshPaymentMethodMetadata struct {
	Name             string  `json:"name" tfsdk:"name"`
	OwnedByWorkspace string  `json:"ownedByWorkspace" tfsdk:"owned_by_workspace"`
	CreatedOn        string  `json:"createdOn" tfsdk:"created_on"`
	DeletedOn        *string `json:"deletedOn" tfsdk:"deleted_on"`
}

type MeshPaymentMethodSpec struct {
	DisplayName    string              `json:"displayName" tfsdk:"display_name"`
	ExpirationDate *string             `json:"expirationDate,omitempty" tfsdk:"expiration_date"`
	Amount         *int64              `json:"amount,omitempty" tfsdk:"amount"`
	Tags           map[string][]string `json:"tags,omitempty" tfsdk:"tags"`
}

type MeshPaymentMethodCreate struct {
	ApiVersion string                          `json:"apiVersion" tfsdk:"api_version"`
	Metadata   MeshPaymentMethodCreateMetadata `json:"metadata" tfsdk:"metadata"`
	Spec       MeshPaymentMethodSpec           `json:"spec" tfsdk:"spec"`
}

type MeshPaymentMethodCreateMetadata struct {
	Name             string `json:"name" tfsdk:"name"`
	OwnedByWorkspace string `json:"ownedByWorkspace" tfsdk:"owned_by_workspace"`
}

type MeshPaymentMethodClient struct {
	meshObject internal.MeshObjectClient[MeshPaymentMethod]
}

func newPaymentMethodClient(httpClient *internal.HttpClient) MeshPaymentMethodClient {
	return MeshPaymentMethodClient{
		meshObject: internal.NewMeshObjectClient[MeshPaymentMethod](httpClient, "v2"),
	}
}

func (c MeshPaymentMethodClient) Read(workspace string, identifier string) (*MeshPaymentMethod, error) {
	return c.meshObject.Get(identifier)
}

func (c MeshPaymentMethodClient) Create(paymentMethod *MeshPaymentMethodCreate) (*MeshPaymentMethod, error) {
	return c.meshObject.Post(paymentMethod)
}

func (c MeshPaymentMethodClient) Update(identifier string, paymentMethod *MeshPaymentMethodCreate) (*MeshPaymentMethod, error) {
	return c.meshObject.Put(identifier, paymentMethod)
}

func (c MeshPaymentMethodClient) Delete(identifier string) error {
	return c.meshObject.Delete(identifier)
}
