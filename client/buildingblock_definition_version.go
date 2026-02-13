package client

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/meshcloud/terraform-provider-meshstack/client/internal"
	"github.com/meshcloud/terraform-provider-meshstack/client/types"
	"github.com/meshcloud/terraform-provider-meshstack/client/types/enum"
)

// Enums

type MeshBuildingBlockDefinitionVersionState string

var (
	MeshBuildingBlockDefinitionVersionStates        = enum.Enum[MeshBuildingBlockDefinitionVersionState]{}
	MeshBuildingBlockDefinitionVersionStateDraft    = MeshBuildingBlockDefinitionVersionStates.Entry("DRAFT")
	MeshBuildingBlockDefinitionVersionStateReleased = MeshBuildingBlockDefinitionVersionStates.Entry("RELEASED")
)

type BuildingBlockDeletionMode string

var (
	BuildingBlockDeletionModes      = enum.Enum[BuildingBlockDeletionMode]{}
	BuildingBlockDeletionModeDelete = BuildingBlockDeletionModes.Entry("DELETE")
	BuildingBlockDeletionModePurge  = BuildingBlockDeletionModes.Entry("PURGE")
)

type MeshBuildingBlockIOType string

var (
	MeshBuildingBlockIOTypes            = enum.Enum[MeshBuildingBlockIOType]{}
	MeshBuildingBlockIOTypeString       = MeshBuildingBlockIOTypes.Entry("STRING")
	MeshBuildingBlockIOTypeCode         = MeshBuildingBlockIOTypes.Entry("CODE")
	MeshBuildingBlockIOTypeInteger      = MeshBuildingBlockIOTypes.Entry("INTEGER")
	MeshBuildingBlockIOTypeBoolean      = MeshBuildingBlockIOTypes.Entry("BOOLEAN")
	MeshBuildingBlockIOTypeFile         = MeshBuildingBlockIOTypes.Entry("FILE")
	MeshBuildingBlockIOTypeList         = MeshBuildingBlockIOTypes.Entry("LIST")
	MeshBuildingBlockIOTypeSingleSelect = MeshBuildingBlockIOTypes.Entry("SINGLE_SELECT")
	MeshBuildingBlockIOTypeMultiSelect  = MeshBuildingBlockIOTypes.Entry("MULTI_SELECT")
)

type MeshBuildingBlockInputAssignmentType string

var (
	MeshBuildingBlockInputAssignmentTypes                           = enum.Enum[MeshBuildingBlockInputAssignmentType]{}
	MeshBuildingBlockInputAssignmentTypeAuthor                      = MeshBuildingBlockInputAssignmentTypes.Entry("AUTHOR")
	MeshBuildingBlockInputAssignmentTypeUserInput                   = MeshBuildingBlockInputAssignmentTypes.Entry("USER_INPUT")
	MeshBuildingBlockInputAssignmentTypePlatformOperatorManualInput = MeshBuildingBlockInputAssignmentTypes.Entry("PLATFORM_OPERATOR_MANUAL_INPUT")
	MeshBuildingBlockInputAssignmentTypeBuildingBlockOutput         = MeshBuildingBlockInputAssignmentTypes.Entry("BUILDING_BLOCK_OUTPUT")
	MeshBuildingBlockInputAssignmentTypePlatformTenantID            = MeshBuildingBlockInputAssignmentTypes.Entry("PLATFORM_TENANT_ID")
	MeshBuildingBlockInputAssignmentTypeWorkspaceIdentifier         = MeshBuildingBlockInputAssignmentTypes.Entry("WORKSPACE_IDENTIFIER")
	MeshBuildingBlockInputAssignmentTypeProjectIdentifier           = MeshBuildingBlockInputAssignmentTypes.Entry("PROJECT_IDENTIFIER")
	MeshBuildingBlockInputAssignmentTypeFullPlatformIdentifier      = MeshBuildingBlockInputAssignmentTypes.Entry("FULL_PLATFORM_IDENTIFIER")
	MeshBuildingBlockInputAssignmentTypeTenantBuildingBlockUuid     = MeshBuildingBlockInputAssignmentTypes.Entry("TENANT_BUILDING_BLOCK_UUID")
	MeshBuildingBlockInputAssignmentTypeStatic                      = MeshBuildingBlockInputAssignmentTypes.Entry("STATIC")
	MeshBuildingBlockInputAssignmentTypeUserPermissions             = MeshBuildingBlockInputAssignmentTypes.Entry("USER_PERMISSIONS")
)

type MeshBuildingBlockDefinitionOutputAssignmentType string

var (
	MeshBuildingBlockDefinitionOutputAssignmentTypes                = enum.Enum[MeshBuildingBlockDefinitionOutputAssignmentType]{}
	MeshBuildingBlockDefinitionOutputAssignmentTypeNone             = MeshBuildingBlockDefinitionOutputAssignmentTypes.Entry("NONE")
	MeshBuildingBlockDefinitionOutputAssignmentTypePlatformTenantID = MeshBuildingBlockDefinitionOutputAssignmentTypes.Entry("PLATFORM_TENANT_ID")
	MeshBuildingBlockDefinitionOutputAssignmentTypeSignInURL        = MeshBuildingBlockDefinitionOutputAssignmentTypes.Entry("SIGN_IN_URL")
	MeshBuildingBlockDefinitionOutputAssignmentTypeResourceURL      = MeshBuildingBlockDefinitionOutputAssignmentTypes.Entry("RESOURCE_URL")
	MeshBuildingBlockDefinitionOutputAssignmentTypeSummary          = MeshBuildingBlockDefinitionOutputAssignmentTypes.Entry("SUMMARY")
)

// Ref types

type BuildingBlockDefinitionRef struct {
	Uuid string `json:"uuid"`
	Kind string `json:"kind"`
}

type MeshIntegrationRef struct {
	Uuid string `json:"uuid" tfsdk:"uuid"`
	Kind string `json:"kind" tfsdk:"kind"`
}

// Input and Output types

type MeshBuildingBlockDefinitionInput struct {
	DisplayName    string                               `json:"displayName" tfsdk:"display_name"`
	Type           MeshBuildingBlockIOType              `json:"type" tfsdk:"type"`
	AssignmentType MeshBuildingBlockInputAssignmentType `json:"assignmentType" tfsdk:"assignment_type"`
	IsEnvironment  bool                                 `json:"isEnvironment" tfsdk:"is_environment"`
	IsSensitive    bool                                 `json:"isSensitive" tfsdk:"-"`
	// If IsSensitive is true, the [types.Variant] (typedef [types.SecretOrAny]) for fields
	// MeshBuildingBlockDefinitionInputAdapter.Argument and
	// MeshBuildingBlockDefinitionInputAdapter.DefaultValue
	// is of [types.Secret] (case [types.Variant.X]).
	// Otherwise, the [types.Variant] is of [types.Any] (case [types.Variant.Y]).
	// As this is a fallback detection when JSON (un)marshaling,
	// types.Any must go second as [types.Variant] intentionally prefers X over Y.
	Argument                    types.SecretOrAny `json:"argument,omitempty" tfsdk:"argument"`
	DefaultValue                types.SecretOrAny `json:"defaultValue,omitempty" tfsdk:"default_value"`
	UpdateableByConsumer        bool              `json:"updateableByConsumer" tfsdk:"updateable_by_consumer"`
	SelectableValues            []types.SetElem   `json:"selectableValues,omitempty" tfsdk:"selectable_values"`
	Description                 *string           `json:"description,omitempty" tfsdk:"description"`
	ValueValidationRegex        *string           `json:"valueValidationRegex,omitempty" tfsdk:"value_validation_regex"`
	ValidationRegexErrorMessage *string           `json:"validationRegexErrorMessage,omitempty" tfsdk:"validation_regex_error_message"`
}

func (m *MeshBuildingBlockDefinitionInput) UnmarshalJSON(bytes []byte) error {
	type wrapped MeshBuildingBlockDefinitionInput
	var target wrapped
	if err := json.Unmarshal(bytes, &target); err != nil {
		return err
	}
	*m = MeshBuildingBlockDefinitionInput(target)
	switch {
	case !m.IsSensitive:
		// ensure "any" struct fields never end up in X accidentally,
		// as X is only set when IsSensitive is true!
		var errs []error
		moveXtoYIfPresent := func(v *types.SecretOrAny) {
			if v.HasX() {
				xJson, err := json.Marshal(v.X)
				errs = append(errs, err)
				v.X = types.Secret{}
				errs = append(errs, json.Unmarshal(xJson, &v.Y))
			}
		}
		moveXtoYIfPresent(&m.Argument)
		moveXtoYIfPresent(&m.DefaultValue)
		return errors.Join(errs...)
	case m.Argument.HasY(), m.DefaultValue.HasY():
		return fmt.Errorf("got sensitive argument or default_value but variant Y is set instead")
	default:
		return nil
	}
}

type MeshBuildingBlockDefinitionOutput struct {
	DisplayName    string                                          `json:"displayName" tfsdk:"display_name"`
	Type           MeshBuildingBlockIOType                         `json:"type" tfsdk:"type"`
	AssignmentType MeshBuildingBlockDefinitionOutputAssignmentType `json:"assignmentType" tfsdk:"assignment_type"`
}

// Main version types

type MeshBuildingBlockDefinitionVersionMetadata struct {
	Uuid             string `json:"uuid"`
	OwnedByWorkspace string `json:"ownedByWorkspace"`
	CreatedOn        string `json:"createdOn"`
}

type BuildingBlockDependencyRef string
type MeshBuildingBlockDefinitionVersionSpec struct {
	BuildingBlockDefinitionRef *BuildingBlockDefinitionRef                  `json:"buildingBlockDefinitionRef" tfsdk:"-"`
	OnlyApplyOncePerTenant     bool                                         `json:"onlyApplyOncePerTenant" tfsdk:"only_apply_once_per_tenant"`
	DeletionMode               BuildingBlockDeletionMode                    `json:"deletionMode" tfsdk:"deletion_mode"`
	Permissions                []ApiPermission                              `json:"permissions,omitempty" tfsdk:"permissions"`
	Outputs                    map[string]MeshBuildingBlockDefinitionOutput `json:"outputs" tfsdk:"outputs"`
	VersionNumber              *int64                                       `json:"versionNumber,omitempty" tfsdk:"version_number"`
	State                      *MeshBuildingBlockDefinitionVersionState     `json:"state,omitempty" tfsdk:"state"`
	RunnerRef                  *BuildingBlockRunnerRef                      `json:"runnerRef" tfsdk:"runner_ref"`
	DependencyDefinitionUUIDs  []BuildingBlockDependencyRef                 `json:"dependencyDefinitionUuids,omitempty" tfsdk:"dependency_refs"`
	Implementation             MeshBuildingBlockDefinitionImplementation    `json:"implementation" tfsdk:"implementation"`
	Inputs                     map[string]*MeshBuildingBlockDefinitionInput `json:"inputs" tfsdk:"inputs"`
}

type MeshBuildingBlockDefinitionVersionStatus struct {
	State      MeshBuildingBlockDefinitionVersionState `json:"state" tfsdk:"state"`
	UsageCount int64                                   `json:"usageCount" tfsdk:"usage_count"`
}

type MeshBuildingBlockDefinitionVersion struct {
	ApiVersion string                                     `json:"apiVersion" tfsdk:"api_version"`
	Kind       string                                     `json:"kind" tfsdk:"kind"`
	Metadata   MeshBuildingBlockDefinitionVersionMetadata `json:"metadata" tfsdk:"metadata"`
	Spec       MeshBuildingBlockDefinitionVersionSpec     `json:"spec" tfsdk:"spec"`
	Status     *MeshBuildingBlockDefinitionVersionStatus  `json:"status,omitempty" tfsdk:"status"`
}

// MeshBuildingBlockDefinitionVersionClient manages a version of a building block definition.
// As such a version is tightly coupled to the definition, there's no single Get or Delete implemented.
// A Get is not required as we always expose all versions of a definition anyway, and a Delete happens together when the definition is deleted.
type MeshBuildingBlockDefinitionVersionClient interface {
	List(ctx context.Context, buildingBlockDefinitionUuid string) ([]MeshBuildingBlockDefinitionVersion, error)
	Create(ctx context.Context, ownedByWorkspace string, versionSpec MeshBuildingBlockDefinitionVersionSpec) (*MeshBuildingBlockDefinitionVersion, error)
	Update(ctx context.Context, uuid, ownedByWorkspace string, versionSpec MeshBuildingBlockDefinitionVersionSpec) (*MeshBuildingBlockDefinitionVersion, error)
}

type meshBuildingBlockDefinitionVersionClient struct {
	meshObject internal.MeshObjectClient[MeshBuildingBlockDefinitionVersion]
}

func newBuildingBlockDefinitionVersionClient(ctx context.Context, httpClient *internal.HttpClient) MeshBuildingBlockDefinitionVersionClient {
	return meshBuildingBlockDefinitionVersionClient{
		meshObject: internal.NewMeshObjectClient[MeshBuildingBlockDefinitionVersion](ctx, httpClient, "v1-preview"),
	}
}

func (c meshBuildingBlockDefinitionVersionClient) List(ctx context.Context, buildingBlockDefinitionUuid string) ([]MeshBuildingBlockDefinitionVersion, error) {
	return c.meshObject.List(ctx, internal.WithUrlQuery("buildingBlockDefinitionUuid", buildingBlockDefinitionUuid))
}

func (c meshBuildingBlockDefinitionVersionClient) Create(ctx context.Context, ownedByWorkspace string, versionSpec MeshBuildingBlockDefinitionVersionSpec) (*MeshBuildingBlockDefinitionVersion, error) {
	return c.meshObject.Post(ctx, MeshBuildingBlockDefinitionVersion{
		ApiVersion: c.meshObject.ApiVersion,
		Kind:       c.meshObject.Kind,
		Metadata: MeshBuildingBlockDefinitionVersionMetadata{
			OwnedByWorkspace: ownedByWorkspace,
		},
		Spec: versionSpec,
	})
}

func (c meshBuildingBlockDefinitionVersionClient) Update(ctx context.Context, uuid, ownedByWorkspace string, versionSpec MeshBuildingBlockDefinitionVersionSpec) (*MeshBuildingBlockDefinitionVersion, error) {
	return c.meshObject.Put(ctx, uuid, MeshBuildingBlockDefinitionVersion{
		ApiVersion: c.meshObject.ApiVersion,
		Kind:       c.meshObject.Kind,
		Metadata: MeshBuildingBlockDefinitionVersionMetadata{
			Uuid:             uuid,
			OwnedByWorkspace: ownedByWorkspace,
		},
		Spec: versionSpec,
	})
}
