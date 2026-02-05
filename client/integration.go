package client

import (
	"context"

	"github.com/meshcloud/terraform-provider-meshstack/client/internal"
	"github.com/meshcloud/terraform-provider-meshstack/client/types"
)

type MeshIntegration struct {
	ApiVersion string                  `json:"apiVersion"`
	Kind       string                  `json:"kind"`
	Metadata   MeshIntegrationMetadata `json:"metadata"`
	Spec       MeshIntegrationSpec     `json:"spec"`
	Status     *MeshIntegrationStatus  `json:"status"`
}

type MeshIntegrationMetadataAdapter[String any] struct {
	Uuid             String `json:"uuid,omitempty" tfsdk:"uuid"`
	OwnedByWorkspace string `json:"ownedByWorkspace" tfsdk:"owned_by_workspace"`
}

type MeshIntegrationMetadata = MeshIntegrationMetadataAdapter[*types.String]

type MeshIntegrationSpec struct {
	DisplayName string                `json:"displayName" tfsdk:"display_name"`
	Config      MeshIntegrationConfig `json:"config" tfsdk:"config"`
}

type MeshIntegrationStatus struct {
	IsBuiltIn                  bool                            `json:"isBuiltIn" tfsdk:"is_built_in"`
	WorkloadIdentityFederation *MeshWorkloadIdentityFederation `json:"workloadIdentityFederation" tfsdk:"workload_identity_federation"`
}

type MeshWorkloadIdentityFederation struct {
	Issuer  string              `json:"issuer" tfsdk:"issuer"`
	Subject string              `json:"subject" tfsdk:"subject"`
	Gcp     *MeshWifProvider    `json:"gcp" tfsdk:"gcp"`
	Aws     *MeshAwsWifProvider `json:"aws" tfsdk:"aws"`
	Azure   *MeshWifProvider    `json:"azure" tfsdk:"azure"`
}

type MeshWifProvider struct {
	Audience string `json:"audience" tfsdk:"audience"`
}

type MeshAwsWifProvider struct {
	Audience   string `json:"audience" tfsdk:"audience"`
	Thumbprint string `json:"thumbprint" tfsdk:"thumbprint"`
}

type MeshIntegrationClient interface {
	Create(ctx context.Context, integration MeshIntegration) (*MeshIntegration, error)
	Read(ctx context.Context, uuid string) (*MeshIntegration, error)
	Update(ctx context.Context, integration MeshIntegration) (*MeshIntegration, error)
	Delete(ctx context.Context, uuid string) error
	List(ctx context.Context) ([]MeshIntegration, error)
}

type meshIntegrationClientImpl struct {
	meshObject internal.MeshObjectClient[MeshIntegration]
}

func newIntegrationClient(ctx context.Context, httpClient *internal.HttpClient) MeshIntegrationClient {
	return &meshIntegrationClientImpl{internal.NewMeshObjectClient[MeshIntegration](ctx, httpClient, "v1-preview")}
}

func (c meshIntegrationClientImpl) Create(ctx context.Context, integration MeshIntegration) (*MeshIntegration, error) {
	integration.Kind = c.meshObject.Kind
	integration.ApiVersion = c.meshObject.ApiVersion
	return c.meshObject.Post(ctx, integration)
}

func (c meshIntegrationClientImpl) Read(ctx context.Context, uuid string) (*MeshIntegration, error) {
	return c.meshObject.Get(ctx, uuid)
}

func (c meshIntegrationClientImpl) Update(ctx context.Context, integration MeshIntegration) (*MeshIntegration, error) {
	integration.Kind = c.meshObject.Kind
	integration.ApiVersion = c.meshObject.ApiVersion
	return c.meshObject.Put(ctx, *integration.Metadata.Uuid, integration)
}

func (c meshIntegrationClientImpl) Delete(ctx context.Context, uuid string) error {
	return c.meshObject.Delete(ctx, uuid)
}

func (c meshIntegrationClientImpl) List(ctx context.Context) ([]MeshIntegration, error) {
	return c.meshObject.List(ctx)
}
