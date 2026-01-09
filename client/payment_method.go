package client

import (
	"net/url"
)

const CONTENT_TYPE_PAYMENT_METHOD = "application/vnd.meshcloud.api.meshpaymentmethod.v2.hal+json"

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

func (c *MeshStackProviderClient) urlForPaymentMethod(identifier string) *url.URL {
	return c.endpoints.PaymentMethods.JoinPath(identifier)
}

func (c *MeshStackProviderClient) ReadPaymentMethod(workspace string, identifier string) (*MeshPaymentMethod, error) {
	return unmarshalBodyIfPresent[MeshPaymentMethod](c.doAuthenticatedRequest("GET", c.urlForPaymentMethod(identifier),
		withAccept(CONTENT_TYPE_PAYMENT_METHOD),
	))
}

func (c *MeshStackProviderClient) CreatePaymentMethod(paymentMethod *MeshPaymentMethodCreate) (*MeshPaymentMethod, error) {
	return unmarshalBody[MeshPaymentMethod](c.doAuthenticatedRequest("POST", c.endpoints.PaymentMethods,
		withPayload(paymentMethod, CONTENT_TYPE_PAYMENT_METHOD),
	))
}

func (c *MeshStackProviderClient) UpdatePaymentMethod(identifier string, paymentMethod *MeshPaymentMethodCreate) (*MeshPaymentMethod, error) {
	return unmarshalBody[MeshPaymentMethod](c.doAuthenticatedRequest("PUT", c.urlForPaymentMethod(identifier),
		withPayload(paymentMethod, CONTENT_TYPE_PAYMENT_METHOD),
	))
}

func (c *MeshStackProviderClient) DeletePaymentMethod(identifier string) error {
	_, err := c.doAuthenticatedRequest("DELETE", c.urlForPaymentMethod(identifier))
	return err
}
