package client

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
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

func (c *MeshStackProviderClient) urlForPaymentMethod(workspace string, identifier string) *url.URL {
	return c.endpoints.PaymentMethods.JoinPath(identifier)
}

func (c *MeshStackProviderClient) ReadPaymentMethod(workspace string, identifier string) (*MeshPaymentMethod, error) {
	targetUrl := c.urlForPaymentMethod(workspace, identifier)

	req, err := http.NewRequest("GET", targetUrl.String(), nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", CONTENT_TYPE_PAYMENT_METHOD)

	res, err := c.doAuthenticatedRequest(req)
	if err != nil {
		return nil, err
	}

	defer res.Body.Close()

	if res.StatusCode == http.StatusNotFound {
		return nil, nil
	}

	data, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}

	if !isSuccessHTTPStatus(res) {
		return nil, fmt.Errorf("unexpected status code: %d, %s", res.StatusCode, data)
	}

	var paymentMethod MeshPaymentMethod
	err = json.Unmarshal(data, &paymentMethod)
	if err != nil {
		return nil, err
	}

	return &paymentMethod, nil
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

	var createdPaymentMethod MeshPaymentMethod
	err = json.Unmarshal(data, &createdPaymentMethod)
	if err != nil {
		return nil, err
	}

	return &createdPaymentMethod, nil
}

func (c *MeshStackProviderClient) UpdatePaymentMethod(workspace string, identifier string, paymentMethod *MeshPaymentMethodCreate) (*MeshPaymentMethod, error) {
	targetUrl := c.urlForPaymentMethod(workspace, identifier)

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

	var updatedPaymentMethod MeshPaymentMethod
	err = json.Unmarshal(data, &updatedPaymentMethod)
	if err != nil {
		return nil, err
	}

	return &updatedPaymentMethod, nil
}

func (c *MeshStackProviderClient) DeletePaymentMethod(workspace string, identifier string) error {
	targetUrl := c.urlForPaymentMethod(workspace, identifier)
	return c.deleteMeshObject(*targetUrl, 204)
}
