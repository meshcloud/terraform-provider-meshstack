package client

import (
	"context"

	"github.com/meshcloud/terraform-provider-meshstack/client/internal"
)

type MeshPaymentMethod struct {
	Metadata MeshPaymentMethodMetadata `json:"metadata" tfsdk:"metadata"`
	Spec     MeshPaymentMethodSpec     `json:"spec" tfsdk:"spec"`
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
	Metadata MeshPaymentMethodCreateMetadata `json:"metadata" tfsdk:"metadata"`
	Spec     MeshPaymentMethodSpec           `json:"spec" tfsdk:"spec"`
}

type MeshPaymentMethodCreateMetadata struct {
	Name             string `json:"name" tfsdk:"name"`
	OwnedByWorkspace string `json:"ownedByWorkspace" tfsdk:"owned_by_workspace"`
}

type MeshPaymentMethodClient interface {
	Read(ctx context.Context, workspace string, identifier string) (*MeshPaymentMethod, error)
	Create(ctx context.Context, paymentMethod *MeshPaymentMethodCreate) (*MeshPaymentMethod, error)
	Update(ctx context.Context, identifier string, paymentMethod *MeshPaymentMethodCreate) (*MeshPaymentMethod, error)
	Delete(ctx context.Context, identifier string) error
}

type meshPaymentMethodClient struct {
	meshObject internal.MeshObjectClient[MeshPaymentMethod]
}

func newPaymentMethodClient(ctx context.Context, httpClient *internal.HttpClient) MeshPaymentMethodClient {
	return meshPaymentMethodClient{internal.NewMeshObjectClient[MeshPaymentMethod](ctx, httpClient, "v2")}
}

func (c meshPaymentMethodClient) Read(ctx context.Context, workspace string, identifier string) (*MeshPaymentMethod, error) {
	return c.meshObject.Get(ctx, identifier)
}

func (c meshPaymentMethodClient) Create(ctx context.Context, paymentMethod *MeshPaymentMethodCreate) (*MeshPaymentMethod, error) {
	return c.meshObject.Post(ctx, paymentMethod)
}

func (c meshPaymentMethodClient) Update(ctx context.Context, identifier string, paymentMethod *MeshPaymentMethodCreate) (*MeshPaymentMethod, error) {
	return c.meshObject.Put(ctx, identifier, paymentMethod)
}

func (c meshPaymentMethodClient) Delete(ctx context.Context, identifier string) error {
	return c.meshObject.Delete(ctx, identifier)
}
