package client

import (
	"time"

	"github.com/meshcloud/terraform-provider-meshstack/client/internal"
)

// Enums

type MeshBuildingBlockType string

const (
	MeshBuildingBlockTypeTenantLevel    MeshBuildingBlockType = "TENANT_LEVEL"
	MeshBuildingBlockTypeWorkspaceLevel MeshBuildingBlockType = "WORKSPACE_LEVEL"
)

type MeshBuildingBlockDefinitionVersionState string

const (
	MeshBuildingBlockDefinitionVersionStateDraft    MeshBuildingBlockDefinitionVersionState = "DRAFT"
	MeshBuildingBlockDefinitionVersionStateReleased MeshBuildingBlockDefinitionVersionState = "RELEASED"
)

// MeshBuildingBlockDefinition types

type MeshBuildingBlockDefinitionMetadata struct {
	UUID                *string             `json:"uuid,omitempty" tfsdk:"uuid"`
	OwnedByWorkspace    string              `json:"ownedByWorkspace" tfsdk:"owned_by_workspace"`
	Tags                map[string][]string `json:"tags,omitempty" tfsdk:"tags"`
	CreatedOn           *time.Time          `json:"createdOn,omitempty" tfsdk:"created_on"`
	MarkedForDeletionOn *time.Time          `json:"markedForDeletionOn,omitempty" tfsdk:"marked_for_deletion_on"`
	MarkedForDeletionBy *string             `json:"markedForDeletionBy,omitempty" tfsdk:"marked_for_deletion_by"`
}

type MeshBuildingBlockDefinitionSpec struct {
	DisplayName                     string                `json:"displayName" tfsdk:"display_name"`
	Symbol                          *string               `json:"symbol,omitempty" tfsdk:"symbol"`
	TargetType                      MeshBuildingBlockType `json:"targetType" tfsdk:"target_type"`
	Description                     string                `json:"description" tfsdk:"description"`
	Readme                          *string               `json:"readme,omitempty" tfsdk:"readme"`
	SupportedPlatforms              []string              `json:"supportedPlatforms" tfsdk:"supported_platforms"`
	RunTransparency                 bool                  `json:"runTransparency" tfsdk:"run_transparency"`
	UseInLandingZonesOnly           bool                  `json:"useInLandingZonesOnly" tfsdk:"use_in_landing_zones_only"`
	SupportURL                      *string               `json:"supportUrl,omitempty" tfsdk:"support_url"`
	DocumentationURL                *string               `json:"documentationUrl,omitempty" tfsdk:"documentation_url"`
	NotificationSubscriberUsernames []string              `json:"notificationSubscriberUsernames" tfsdk:"notification_subscriber_usernames"`
}

type MeshBuildingBlockDefinitionStatusVersion struct {
	VersionUUID   string                                  `json:"versionUuid" tfsdk:"version_uuid"`
	VersionNumber int64                                   `json:"versionNumber" tfsdk:"version_number"`
	State         MeshBuildingBlockDefinitionVersionState `json:"state" tfsdk:"state"`
}

type MeshBuildingBlockDefinitionStatus struct {
	UsageCount                *int64                                     `json:"usageCount,omitempty" tfsdk:"usage_count"`
	Versions                  []MeshBuildingBlockDefinitionStatusVersion `json:"versions" tfsdk:"versions"`
	LatestVersion             int64                                      `json:"latestVersion" tfsdk:"latest_version"`
	LatestVersionUUID         string                                     `json:"latestVersionUuid" tfsdk:"latest_version_uuid"`
	LatestReleasedVersion     *int64                                     `json:"latestReleasedVersion,omitempty" tfsdk:"latest_released_version"`
	LatestReleasedVersionUUID *string                                    `json:"latestReleasedVersionUuid,omitempty" tfsdk:"latest_released_version_uuid"`
}

type MeshBuildingBlockDefinition struct {
	ApiVersion string                              `json:"apiVersion" tfsdk:"api_version"`
	Kind       string                              `json:"kind" tfsdk:"kind"`
	Metadata   MeshBuildingBlockDefinitionMetadata `json:"metadata" tfsdk:"metadata"`
	Spec       MeshBuildingBlockDefinitionSpec     `json:"spec" tfsdk:"spec"`
	Status     *MeshBuildingBlockDefinitionStatus  `json:"status,omitempty" tfsdk:"status"`
}

type MeshBuildingBlockDefinitionClient struct {
	meshObject internal.MeshObjectClient[MeshBuildingBlockDefinition]
}

func newBuildingBlockDefinitionClient(httpClient *internal.HttpClient) MeshBuildingBlockDefinitionClient {
	return MeshBuildingBlockDefinitionClient{
		meshObject: internal.NewMeshObjectClient[MeshBuildingBlockDefinition](httpClient, "v1-preview"),
	}
}

func (c MeshBuildingBlockDefinitionClient) Read(uuid string) (*MeshBuildingBlockDefinition, error) {
	return c.meshObject.Get(uuid)
}
