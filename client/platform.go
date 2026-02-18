package client

import (
	"context"

	"github.com/meshcloud/terraform-provider-meshstack/client/internal"
	clientTypes "github.com/meshcloud/terraform-provider-meshstack/client/types"
)

type MeshPlatform struct {
	ApiVersion string               `json:"apiVersion" tfsdk:"-"`
	Kind       string               `json:"kind" tfsdk:"-"`
	Metadata   MeshPlatformMetadata `json:"metadata" tfsdk:"metadata"`
	Spec       MeshPlatformSpec     `json:"spec" tfsdk:"spec"`
}

type MeshPlatformMetadata struct {
	Name             string  `json:"name" tfsdk:"name"`
	OwnedByWorkspace string  `json:"ownedByWorkspace" tfsdk:"owned_by_workspace"`
	Uuid             *string `json:"uuid,omitempty" tfsdk:"uuid"`
}

type MeshPlatformSpec struct {
	DisplayName            string                `json:"displayName" tfsdk:"display_name"`
	Description            string                `json:"description" tfsdk:"description"`
	Endpoint               string                `json:"endpoint" tfsdk:"endpoint"`
	SupportUrl             *string               `json:"supportUrl,omitempty" tfsdk:"support_url"`
	DocumentationUrl       *string               `json:"documentationUrl,omitempty" tfsdk:"documentation_url"`
	LocationRef            LocationRef           `json:"locationRef" tfsdk:"location_ref"`
	ContributingWorkspaces []clientTypes.SetElem `json:"contributingWorkspaces" tfsdk:"contributing_workspaces"`
	Availability           PlatformAvailability  `json:"availability" tfsdk:"availability"`
	Config                 PlatformConfig        `json:"config" tfsdk:"config"`
	QuotaDefinitions       []QuotaDefinition     `json:"quotaDefinitions" tfsdk:"quota_definitions"`
}

type QuotaDefinition struct {
	QuotaKey              string `json:"quotaKey" tfsdk:"quota_key"`
	MinValue              int64  `json:"minValue" tfsdk:"min_value"`
	MaxValue              int64  `json:"maxValue" tfsdk:"max_value"`
	Unit                  string `json:"unit" tfsdk:"unit"`
	AutoApprovalThreshold int64  `json:"autoApprovalThreshold" tfsdk:"auto_approval_threshold"`
	Description           string `json:"description" tfsdk:"description"`
	Label                 string `json:"label" tfsdk:"label"`
}

type LocationRef struct {
	Kind string `json:"kind" tfsdk:"kind"`
	Name string `json:"name" tfsdk:"name"`
}

type PlatformAvailability struct {
	Restriction            string                `json:"restriction" tfsdk:"restriction"`
	PublicationState       string                `json:"publicationState" tfsdk:"publication_state"`
	RestrictedToWorkspaces []clientTypes.SetElem `json:"restrictedToWorkspaces,omitempty" tfsdk:"restricted_to_workspaces"`
}

type PlatformConfig struct {
	Type       string                    `json:"type" tfsdk:"type"`
	Custom     *CustomPlatformConfig     `json:"custom,omitempty" tfsdk:"custom"`
	Aws        *AwsPlatformConfig        `json:"aws,omitempty" tfsdk:"aws"`
	Aks        *AksPlatformConfig        `json:"aks,omitempty" tfsdk:"aks"`
	Azure      *AzurePlatformConfig      `json:"azure,omitempty" tfsdk:"azure"`
	AzureRg    *AzureRgPlatformConfig    `json:"azurerg,omitempty" tfsdk:"azurerg"`
	Gcp        *GcpPlatformConfig        `json:"gcp,omitempty" tfsdk:"gcp"`
	Kubernetes *KubernetesPlatformConfig `json:"kubernetes,omitempty" tfsdk:"kubernetes"`
	OpenShift  *OpenShiftPlatformConfig  `json:"openshift,omitempty" tfsdk:"openshift"`
}

type MeshPlatformMeteringProcessingConfig struct {
	CompactTimelinesAfterDays int64 `json:"compactTimelinesAfterDays" tfsdk:"compact_timelines_after_days"`
	DeleteRawDataAfterDays    int64 `json:"deleteRawDataAfterDays" tfsdk:"delete_raw_data_after_days"`
}

type MeshTenantTags struct {
	NamespacePrefix string      `json:"namespacePrefix" tfsdk:"namespace_prefix"`
	TagMappers      []TagMapper `json:"tagMappers" tfsdk:"tag_mappers"`
}

type TagMapper struct {
	Key          string `json:"key" tfsdk:"key"`
	ValuePattern string `json:"valuePattern" tfsdk:"value_pattern"`
}

type MeshPlatformClient interface {
	Read(ctx context.Context, uuid string) (*MeshPlatform, error)
	Create(ctx context.Context, platform MeshPlatform) (*MeshPlatform, error)
	Update(ctx context.Context, uuid string, platform MeshPlatform) (*MeshPlatform, error)
	Delete(ctx context.Context, uuid string) error
}

type meshPlatformClient struct {
	meshObject internal.MeshObjectClient[MeshPlatform]
}

func newPlatformClient(ctx context.Context, httpClient *internal.HttpClient) MeshPlatformClient {
	return meshPlatformClient{internal.NewMeshObjectClient[MeshPlatform](ctx, httpClient, "v2")}
}

func (c meshPlatformClient) Read(ctx context.Context, uuid string) (*MeshPlatform, error) {
	return c.meshObject.Get(ctx, uuid)
}

func (c meshPlatformClient) Create(ctx context.Context, platform MeshPlatform) (*MeshPlatform, error) {
	platform.Kind = c.meshObject.Kind
	platform.ApiVersion = c.meshObject.ApiVersion
	return c.meshObject.Post(ctx, platform)
}

func (c meshPlatformClient) Update(ctx context.Context, uuid string, platform MeshPlatform) (*MeshPlatform, error) {
	platform.Kind = c.meshObject.Kind
	platform.ApiVersion = c.meshObject.ApiVersion
	return c.meshObject.Put(ctx, uuid, platform)
}

func (c meshPlatformClient) Delete(ctx context.Context, uuid string) error {
	return c.meshObject.Delete(ctx, uuid)
}
