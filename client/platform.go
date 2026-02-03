package client

import (
	"context"

	"github.com/meshcloud/terraform-provider-meshstack/client/internal"
)

type MeshPlatform struct {
	ApiVersion string               `json:"apiVersion" tfsdk:"api_version"`
	Kind       string               `json:"kind" tfsdk:"kind"`
	Metadata   MeshPlatformMetadata `json:"metadata" tfsdk:"metadata"`
	Spec       MeshPlatformSpec     `json:"spec" tfsdk:"spec"`
}

type MeshPlatformMetadata struct {
	Name             string  `json:"name" tfsdk:"name"`
	OwnedByWorkspace string  `json:"ownedByWorkspace" tfsdk:"owned_by_workspace"`
	Uuid             string  `json:"uuid" tfsdk:"uuid"`
	CreatedOn        string  `json:"createdOn" tfsdk:"created_on"`
	DeletedOn        *string `json:"deletedOn" tfsdk:"deleted_on"`
}

type MeshPlatformSpec struct {
	DisplayName            string               `json:"displayName" tfsdk:"display_name"`
	Description            string               `json:"description" tfsdk:"description"`
	Endpoint               string               `json:"endpoint" tfsdk:"endpoint"`
	SupportUrl             *string              `json:"supportUrl,omitempty" tfsdk:"support_url"`
	DocumentationUrl       *string              `json:"documentationUrl,omitempty" tfsdk:"documentation_url"`
	LocationRef            LocationRef          `json:"locationRef" tfsdk:"location_ref"`
	ContributingWorkspaces []string             `json:"contributingWorkspaces" tfsdk:"contributing_workspaces"`
	Availability           PlatformAvailability `json:"availability" tfsdk:"availability"`
	Config                 PlatformConfig       `json:"config" tfsdk:"config"`
	QuotaDefinitions       []QuotaDefinition    `json:"quotaDefinitions" tfsdk:"quota_definitions"`
}

type SecretEmbedded struct {
	Plaintext *string `json:"plaintext,omitempty" tfsdk:"plaintext"`
	// TODO: add Hash field
}

type QuotaDefinition struct {
	QuotaKey              string `json:"quotaKey" tfsdk:"quota_key"`
	MinValue              int    `json:"minValue" tfsdk:"min_value"`
	MaxValue              int    `json:"maxValue" tfsdk:"max_value"`
	Unit                  string `json:"unit" tfsdk:"unit"`
	AutoApprovalThreshold int    `json:"autoApprovalThreshold" tfsdk:"auto_approval_threshold"`
	Description           string `json:"description" tfsdk:"description"`
	Label                 string `json:"label" tfsdk:"label"`
}

type LocationRef struct {
	Kind string `json:"kind" tfsdk:"kind"`
	Name string `json:"name" tfsdk:"name"`
}

type PlatformAvailability struct {
	Restriction            string   `json:"restriction" tfsdk:"restriction"`
	PublicationState       string   `json:"publicationState" tfsdk:"publication_state"`
	RestrictedToWorkspaces []string `json:"restrictedToWorkspaces,omitempty" tfsdk:"restricted_to_workspaces"`
}

type PlatformConfig struct {
	Type       string                    `json:"type" tfsdk:"type"`
	Aws        *AwsPlatformConfig        `json:"aws,omitempty" tfsdk:"aws"`
	Aks        *AksPlatformConfig        `json:"aks,omitempty" tfsdk:"aks"`
	Azure      *AzurePlatformConfig      `json:"azure,omitempty" tfsdk:"azure"`
	AzureRg    *AzureRgPlatformConfig    `json:"azurerg,omitempty" tfsdk:"azurerg"`
	Gcp        *GcpPlatformConfig        `json:"gcp,omitempty" tfsdk:"gcp"`
	Kubernetes *KubernetesPlatformConfig `json:"kubernetes,omitempty" tfsdk:"kubernetes"`
	OpenShift  *OpenShiftPlatformConfig  `json:"openshift,omitempty" tfsdk:"openshift"`
}

type MeshPlatformCreate struct {
	ApiVersion string                     `json:"apiVersion" tfsdk:"api_version"`
	Metadata   MeshPlatformCreateMetadata `json:"metadata" tfsdk:"metadata"`
	Spec       MeshPlatformSpec           `json:"spec" tfsdk:"spec"`
}

type MeshPlatformCreateMetadata struct {
	Name             string `json:"name" tfsdk:"name"`
	OwnedByWorkspace string `json:"ownedByWorkspace" tfsdk:"owned_by_workspace"`
}

type MeshPlatformUpdate struct {
	ApiVersion string                     `json:"apiVersion" tfsdk:"api_version"`
	Metadata   MeshPlatformUpdateMetadata `json:"metadata" tfsdk:"metadata"`
	Spec       MeshPlatformSpec           `json:"spec" tfsdk:"spec"`
}

type MeshPlatformUpdateMetadata struct {
	Name             string `json:"name" tfsdk:"name"`
	OwnedByWorkspace string `json:"ownedByWorkspace" tfsdk:"owned_by_workspace"`
	Uuid             string `json:"uuid" tfsdk:"uuid"`
}

type MeshPlatformMeteringProcessingConfig struct {
	CompactTimelinesAfterDays *int64 `json:"compactTimelinesAfterDays,omitempty" tfsdk:"compact_timelines_after_days"`
	DeleteRawDataAfterDays    *int64 `json:"deleteRawDataAfterDays,omitempty" tfsdk:"delete_raw_data_after_days"`
}

type MeshTenantTags struct {
	NamespacePrefix string      `json:"namespacePrefix" tfsdk:"namespace_prefix"`
	TagMappers      []TagMapper `json:"tagMappers" tfsdk:"tag_mappers"`
}

type TagMapper struct {
	Key          string `json:"key" tfsdk:"key"`
	ValuePattern string `json:"valuePattern" tfsdk:"value_pattern"`
}

type MeshPlatformClient struct {
	meshObject internal.MeshObjectClient[MeshPlatform]
}

func newPlatformClient(ctx context.Context, httpClient *internal.HttpClient) MeshPlatformClient {
	return MeshPlatformClient{internal.NewMeshObjectClient[MeshPlatform](ctx, httpClient, "v2-preview")}
}

func (c MeshPlatformClient) Read(ctx context.Context, uuid string) (*MeshPlatform, error) {
	return c.meshObject.Get(ctx, uuid)
}

func (c MeshPlatformClient) Create(ctx context.Context, platform *MeshPlatformCreate) (*MeshPlatform, error) {
	return c.meshObject.Post(ctx, platform)
}

func (c MeshPlatformClient) Update(ctx context.Context, uuid string, platform *MeshPlatformUpdate) (*MeshPlatform, error) {
	return c.meshObject.Put(ctx, uuid, platform)
}

func (c MeshPlatformClient) Delete(ctx context.Context, uuid string) error {
	return c.meshObject.Delete(ctx, uuid)
}
