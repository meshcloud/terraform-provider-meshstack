package client

import (
	"context"

	"github.com/meshcloud/terraform-provider-meshstack/client/internal"
)

type MeshPlatformType struct {
	ApiVersion string                   `json:"apiVersion" tfsdk:"api_version"`
	Kind       string                   `json:"kind" tfsdk:"kind"`
	Metadata   MeshPlatformTypeMetadata `json:"metadata" tfsdk:"metadata"`
	Spec       MeshPlatformTypeSpec     `json:"spec" tfsdk:"spec"`
	Status     MeshPlatformTypeStatus   `json:"status" tfsdk:"status"`
}

type MeshPlatformTypeStatus struct {
	Lifecycle MeshPlatformTypeLifecycle `json:"lifecycle" tfsdk:"lifecycle"`
}

type MeshPlatformTypeLifecycle struct {
	State string `json:"state" tfsdk:"state"`
}

type MeshPlatformTypeMetadata struct {
	Name             string  `json:"name" tfsdk:"name"`
	OwnedByWorkspace string  `json:"ownedByWorkspace" tfsdk:"owned_by_workspace"`
	CreatedOn        *string `json:"createdOn" tfsdk:"created_on"`
	Uuid             *string `json:"uuid,omitempty" tfsdk:"uuid"`
}

type MeshPlatformTypeSpec struct {
	DisplayName     string  `json:"displayName" tfsdk:"display_name"`
	Category        string  `json:"category" tfsdk:"category"`
	DefaultEndpoint *string `json:"defaultEndpoint,omitempty" tfsdk:"default_endpoint"`
	Icon            string  `json:"icon" tfsdk:"icon"`
}

type MeshPlatformTypeCreate struct {
	ApiVersion string                         `json:"apiVersion" tfsdk:"api_version"`
	Kind       string                         `json:"kind" tfsdk:"kind"`
	Metadata   MeshPlatformTypeCreateMetadata `json:"metadata" tfsdk:"metadata"`
	Spec       MeshPlatformTypeSpec           `json:"spec" tfsdk:"spec"`
}

type MeshPlatformTypeCreateMetadata struct {
	Name             string `json:"name" tfsdk:"name"`
	OwnedByWorkspace string `json:"ownedByWorkspace" tfsdk:"owned_by_workspace"`
}

type MeshPlatformTypeClient struct {
	meshObject internal.MeshObjectClient[MeshPlatformType]
}

func newPlatformTypeClient(ctx context.Context, httpClient *internal.HttpClient) MeshPlatformTypeClient {
	return MeshPlatformTypeClient{internal.NewMeshObjectClient[MeshPlatformType](ctx, httpClient, "v1-preview")}
}

func (c MeshPlatformTypeClient) Read(ctx context.Context, identifier string) (*MeshPlatformType, error) {
	return c.meshObject.Get(ctx, identifier)
}

func (c MeshPlatformTypeClient) Create(ctx context.Context, platformType *MeshPlatformTypeCreate) (*MeshPlatformType, error) {
	return c.meshObject.Post(ctx, platformType)
}

func (c MeshPlatformTypeClient) Update(ctx context.Context, name string, platformType *MeshPlatformTypeCreate) (*MeshPlatformType, error) {
	return c.meshObject.Put(ctx, name, platformType)
}

func (c MeshPlatformTypeClient) Delete(ctx context.Context, name string) error {
	return c.meshObject.Delete(ctx, name)
}

func (c MeshPlatformTypeClient) List(ctx context.Context, category *string, lifecycleStatus *string) ([]MeshPlatformType, error) {
	var options []internal.RequestOption
	if category != nil {
		options = append(options, internal.WithUrlQuery("category", *category))
	}
	if lifecycleStatus != nil {
		options = append(options, internal.WithUrlQuery("lifecycleStatus", *lifecycleStatus))
	}
	return c.meshObject.List(ctx, options...)
}
