package client

import (
	"bytes"
	"encoding/json"
	"net/http"
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
	targetUrl := c.urlForPaymentMethod(identifier)

	req, err := http.NewRequest("GET", targetUrl.String(), nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", CONTENT_TYPE_PAYMENT_METHOD)

	return unmarshalBodyIfPresent[MeshPaymentMethod](c.doAuthenticatedRequest(req))
}

func (c *MeshStackProviderClient) CreatePaymentMethod(paymentMethod *MeshPaymentMethodCreate) (*MeshPaymentMethod, error) {
	payload, err := json.Marshal(paymentMethod)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest("POST", c.endpoints.PaymentMethods.String(), bytes.NewBuffer(payload))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", CONTENT_TYPE_PAYMENT_METHOD)
	req.Header.Set("Accept", CONTENT_TYPE_PAYMENT_METHOD)

	return unmarshalBody[MeshPaymentMethod](c.doAuthenticatedRequest(req))
}

func (c *MeshStackProviderClient) UpdatePaymentMethod(identifier string, paymentMethod *MeshPaymentMethodCreate) (*MeshPaymentMethod, error) {
	targetUrl := c.urlForPaymentMethod(identifier)

	payload, err := json.Marshal(paymentMethod)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest("PUT", targetUrl.String(), bytes.NewBuffer(payload))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", CONTENT_TYPE_PAYMENT_METHOD)
	req.Header.Set("Accept", CONTENT_TYPE_PAYMENT_METHOD)

	return unmarshalBody[MeshPaymentMethod](c.doAuthenticatedRequest(req))
}

func (c *MeshStackProviderClient) DeletePaymentMethod(identifier string) error {
	targetUrl := c.urlForPaymentMethod(identifier)
	return c.deleteMeshObject(*targetUrl, 204)
}
