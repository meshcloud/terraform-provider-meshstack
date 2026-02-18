package client

import (
	"context"

	"github.com/meshcloud/terraform-provider-meshstack/client/internal"
)

type MeshLandingZone struct {
	ApiVersion string                  `json:"apiVersion" tfsdk:"-"`
	Kind       string                  `json:"kind" tfsdk:"-"`
	Metadata   MeshLandingZoneMetadata `json:"metadata" tfsdk:"metadata"`
	Spec       MeshLandingZoneSpec     `json:"spec" tfsdk:"spec"`
	Status     MeshLandingZoneStatus   `json:"status" tfsdk:"status"`
}

type MeshLandingZoneMetadata struct {
	Name             string              `json:"name" tfsdk:"name"`
	OwnedByWorkspace string              `json:"ownedByWorkspace" tfsdk:"owned_by_workspace"`
	Tags             map[string][]string `json:"tags" tfsdk:"tags"`
}

type MeshLandingZoneSpec struct {
	DisplayName                  string                             `json:"displayName" tfsdk:"display_name"`
	Description                  string                             `json:"description" tfsdk:"description"`
	AutomateDeletionApproval     bool                               `json:"automateDeletionApproval" tfsdk:"automate_deletion_approval"`
	AutomateDeletionReplication  bool                               `json:"automateDeletionReplication" tfsdk:"automate_deletion_replication"`
	InfoLink                     *string                            `json:"infoLink,omitempty" tfsdk:"info_link"`
	PlatformRef                  MeshLandingZonePlatformRef         `json:"platformRef" tfsdk:"platform_ref"`
	PlatformProperties           *MeshLandingZonePlatformProperties `json:"platformProperties,omitempty" tfsdk:"platform_properties"`
	Quotas                       []MeshLandingZoneQuota             `json:"quotas" tfsdk:"quotas"`
	MandatoryBuildingBlockRefs   []MeshBuildingBlockDefinitionRef   `json:"mandatoryBuildingBlockRefs" tfsdk:"mandatory_building_block_refs"`
	RecommendedBuildingBlockRefs []MeshBuildingBlockDefinitionRef   `json:"recommendedBuildingBlockRefs" tfsdk:"recommended_building_block_refs"`
}

type MeshLandingZoneStatus struct {
	Disabled   bool `json:"disabled" tfsdk:"disabled"`
	Restricted bool `json:"restricted" tfsdk:"restricted"`
}

type MeshLandingZonePlatformRef struct {
	Uuid string `json:"uuid" tfsdk:"uuid"`
	Kind string `json:"kind" tfsdk:"kind"`
}

type MeshLandingZonePlatformProperties struct {
	Type       string                        `json:"type" tfsdk:"type"`
	Aws        *AwsPlatformProperties        `json:"aws" tfsdk:"aws"`
	Aks        *AksPlatformProperties        `json:"aks" tfsdk:"aks"`
	Azure      *AzurePlatformProperties      `json:"azure" tfsdk:"azure"`
	AzureRg    *AzureRgPlatformProperties    `json:"azurerg" tfsdk:"azurerg"`
	Custom     *CustomPlatformProperties     `json:"custom" tfsdk:"custom"`
	Gcp        *GcpPlatformProperties        `json:"gcp" tfsdk:"gcp"`
	Kubernetes *KubernetesPlatformProperties `json:"kubernetes" tfsdk:"kubernetes"`
	OpenShift  *OpenShiftPlatformProperties  `json:"openshift" tfsdk:"openshift"`
}

type MeshLandingZoneQuota struct {
	Key   string `json:"key" tfsdk:"key"`
	Value int64  `json:"value" tfsdk:"value"`
}

type MeshLandingZoneCreate struct {
	ApiVersion string                  `json:"apiVersion" tfsdk:"-"`
	Metadata   MeshLandingZoneMetadata `json:"metadata" tfsdk:"metadata"`
	Spec       MeshLandingZoneSpec     `json:"spec" tfsdk:"spec"`
}

type MeshLandingZoneClient struct {
	meshObject internal.MeshObjectClient[MeshLandingZone]
}

func newLandingZoneClient(ctx context.Context, httpClient *internal.HttpClient) MeshLandingZoneClient {
	return MeshLandingZoneClient{internal.NewMeshObjectClient[MeshLandingZone](ctx, httpClient, "v1")}
}

func (c MeshLandingZoneClient) Read(ctx context.Context, name string) (*MeshLandingZone, error) {
	return c.meshObject.Get(ctx, name)
}

func (c MeshLandingZoneClient) Create(ctx context.Context, landingZone *MeshLandingZoneCreate) (*MeshLandingZone, error) {
	return c.meshObject.Post(ctx, landingZone)
}

func (c MeshLandingZoneClient) Update(ctx context.Context, name string, landingZone *MeshLandingZoneCreate) (*MeshLandingZone, error) {
	return c.meshObject.Put(ctx, name, landingZone)
}

func (c MeshLandingZoneClient) Delete(ctx context.Context, name string) error {
	return c.meshObject.Delete(ctx, name)
}
