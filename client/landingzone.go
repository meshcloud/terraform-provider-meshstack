package client

import (
	"net/url"
)

const CONTENT_TYPE_LANDINGZONE = "application/vnd.meshcloud.api.meshlandingzone.v1-preview.hal+json"

type MeshLandingZone struct {
	ApiVersion string                  `json:"apiVersion" tfsdk:"api_version"`
	Kind       string                  `json:"kind" tfsdk:"kind"`
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
	Gcp        *GcpPlatformProperties        `json:"gcp" tfsdk:"gcp"`
	Kubernetes *KubernetesPlatformProperties `json:"kubernetes" tfsdk:"kubernetes"`
	OpenShift  *OpenShiftPlatformProperties  `json:"openshift" tfsdk:"openshift"`
}

type MeshLandingZoneQuota struct {
	Key   string `json:"key" tfsdk:"key"`
	Value int64  `json:"value" tfsdk:"value"`
}

type MeshLandingZoneCreate struct {
	ApiVersion string                  `json:"apiVersion" tfsdk:"api_version"`
	Metadata   MeshLandingZoneMetadata `json:"metadata" tfsdk:"metadata"`
	Spec       MeshLandingZoneSpec     `json:"spec" tfsdk:"spec"`
}

func (c *MeshStackProviderClient) urlForLandingZone(name string) *url.URL {
	return c.endpoints.LandingZones.JoinPath(name)
}

func (c *MeshStackProviderClient) ReadLandingZone(name string) (*MeshLandingZone, error) {
	return unmarshalBodyIfPresent[MeshLandingZone](c.doAuthenticatedRequest("GET", c.urlForLandingZone(name),
		withAccept(CONTENT_TYPE_LANDINGZONE),
	))
}

func (c *MeshStackProviderClient) CreateLandingZone(landingZone *MeshLandingZoneCreate) (*MeshLandingZone, error) {
	return unmarshalBody[MeshLandingZone](c.doAuthenticatedRequest("POST", c.endpoints.LandingZones,
		withPayload(landingZone, CONTENT_TYPE_LANDINGZONE),
	))
}

func (c *MeshStackProviderClient) UpdateLandingZone(name string, landingZone *MeshLandingZoneCreate) (*MeshLandingZone, error) {
	return unmarshalBody[MeshLandingZone](c.doAuthenticatedRequest("PUT", c.urlForLandingZone(name),
		withPayload(landingZone, CONTENT_TYPE_LANDINGZONE),
	))
}

func (c *MeshStackProviderClient) DeleteLandingZone(name string) error {
	_, err := c.doAuthenticatedRequest("DELETE", c.urlForLandingZone(name))
	return err
}
