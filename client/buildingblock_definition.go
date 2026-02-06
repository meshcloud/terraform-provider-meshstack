package client

import (
	"context"

	"github.com/meshcloud/terraform-provider-meshstack/client/internal"
	"github.com/meshcloud/terraform-provider-meshstack/client/types"
	"github.com/meshcloud/terraform-provider-meshstack/client/types/enum"
)

// Enums

type MeshBuildingBlockType string

var (
	MeshBuildingBlockTypes              = enum.Enum[MeshBuildingBlockType]{}
	MeshBuildingBlockTypeTenantLevel    = MeshBuildingBlockTypes.Entry("TENANT_LEVEL")
	MeshBuildingBlockTypeWorkspaceLevel = MeshBuildingBlockTypes.Entry("WORKSPACE_LEVEL")
)

// MeshBuildingBlockDefinition types

type MeshBuildingBlockDefinitionMetadataBase struct {
	OwnedByWorkspace string              `json:"ownedByWorkspace" tfsdk:"owned_by_workspace"`
	Tags             map[string][]string `json:"tags" tfsdk:"tags"`
}

type MeshBuildingBlockDefinitionMetadataAdapter[String any] struct {
	MeshBuildingBlockDefinitionMetadataBase
	Uuid String `json:"uuid,omitempty" tfsdk:"uuid"`
}

type MeshBuildingBlockDefinitionMetadata = MeshBuildingBlockDefinitionMetadataAdapter[*types.String]

type MeshBuildingBlockDefinitionSpecBase struct {
	DisplayName           string                `json:"displayName" tfsdk:"display_name"`
	TargetType            MeshBuildingBlockType `json:"targetType" tfsdk:"target_type"`
	Description           string                `json:"description" tfsdk:"description"`
	Readme                *string               `json:"readme,omitempty" tfsdk:"readme"`
	RunTransparency       bool                  `json:"runTransparency" tfsdk:"run_transparency"`
	UseInLandingZonesOnly bool                  `json:"useInLandingZonesOnly" tfsdk:"use_in_landing_zones_only"`
	SupportURL            *string               `json:"supportUrl,omitempty" tfsdk:"support_url"`
	DocumentationURL      *string               `json:"documentationUrl,omitempty" tfsdk:"documentation_url"`
	// Note: You can also specify emails with prefix 'email:', so it's not only usernames!
	NotificationSubscribers []string `json:"notificationSubscriberUsernames,omitempty" tfsdk:"notification_subscribers"`
}

type MeshBuildingBlockDefinitionSpecAdapter[String, SupportedPlatformRef any] struct {
	MeshBuildingBlockDefinitionSpecBase
	Symbol             String                 `json:"symbol,omitempty" tfsdk:"symbol"`
	SupportedPlatforms []SupportedPlatformRef `json:"supportedPlatforms" tfsdk:"supported_platforms"`
}

type MeshBuildingBlockDefinitionSpec = MeshBuildingBlockDefinitionSpecAdapter[*types.String, types.String]

type MeshBuildingBlockDefinitionStatusVersion struct {
	VersionUuid   string                                  `json:"versionUuid"`
	VersionNumber int64                                   `json:"versionNumber" `
	State         MeshBuildingBlockDefinitionVersionState `json:"state"`
}

type MeshBuildingBlockDefinitionStatus struct {
	UsageCount                *int64                                     `json:"usageCount"`
	Versions                  []MeshBuildingBlockDefinitionStatusVersion `json:"versions"`
	LatestVersion             int64                                      `json:"latestVersion"`
	LatestVersionUuid         string                                     `json:"latestVersionUuid" `
	LatestReleasedVersion     *int64                                     `json:"latestReleasedVersion"`
	LatestReleasedVersionUuid *string                                    `json:"latestReleasedVersionUuid"`
}

type MeshBuildingBlockDefinition struct {
	ApiVersion string                              `json:"apiVersion"`
	Kind       string                              `json:"kind"`
	Metadata   MeshBuildingBlockDefinitionMetadata `json:"metadata" `
	Spec       MeshBuildingBlockDefinitionSpec     `json:"spec"`
	Status     *MeshBuildingBlockDefinitionStatus  `json:"status,omitempty"`
}

type MeshBuildingBlockDefinitionClient interface {
	List(ctx context.Context, workspaceIdentifier *string) ([]MeshBuildingBlockDefinition, error)
	Read(ctx context.Context, uuid string) (*MeshBuildingBlockDefinition, error)
	Create(ctx context.Context, definition MeshBuildingBlockDefinition) (*MeshBuildingBlockDefinition, error)
	Update(ctx context.Context, uuid string, definition MeshBuildingBlockDefinition) (*MeshBuildingBlockDefinition, error)
	Delete(ctx context.Context, uuid string) error
}

type meshBuildingBlockDefinitionClient struct {
	meshObject internal.MeshObjectClient[MeshBuildingBlockDefinition]
}

func newBuildingBlockDefinitionClient(ctx context.Context, httpClient *internal.HttpClient) MeshBuildingBlockDefinitionClient {
	return meshBuildingBlockDefinitionClient{
		meshObject: internal.NewMeshObjectClient[MeshBuildingBlockDefinition](ctx, httpClient, "v1-preview"),
	}
}

func (c meshBuildingBlockDefinitionClient) List(ctx context.Context, workspaceIdentifier *string) ([]MeshBuildingBlockDefinition, error) {
	var options []internal.RequestOption
	if workspaceIdentifier != nil {
		options = append(options, internal.WithUrlQuery("workspaceIdentifier", *workspaceIdentifier))
	}
	return c.meshObject.List(ctx, options...)
}

func (c meshBuildingBlockDefinitionClient) Read(ctx context.Context, uuid string) (*MeshBuildingBlockDefinition, error) {
	return c.meshObject.Get(ctx, uuid)
}

func (c meshBuildingBlockDefinitionClient) Create(ctx context.Context, definition MeshBuildingBlockDefinition) (*MeshBuildingBlockDefinition, error) {
	definition.Kind = c.meshObject.Kind
	definition.ApiVersion = c.meshObject.ApiVersion
	return c.meshObject.Post(ctx, definition)
}

func (c meshBuildingBlockDefinitionClient) Update(ctx context.Context, uuid string, definition MeshBuildingBlockDefinition) (*MeshBuildingBlockDefinition, error) {
	definition.Kind = c.meshObject.Kind
	definition.ApiVersion = c.meshObject.ApiVersion
	return c.meshObject.Put(ctx, uuid, definition)
}

func (c meshBuildingBlockDefinitionClient) Delete(ctx context.Context, uuid string) error {
	return c.meshObject.Delete(ctx, uuid)
}
