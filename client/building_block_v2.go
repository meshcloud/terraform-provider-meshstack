package client

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"slices"

	"github.com/meshcloud/terraform-provider-meshstack/client/internal"
	"github.com/meshcloud/terraform-provider-meshstack/client/types"
	"github.com/meshcloud/terraform-provider-meshstack/client/types/enum"
)

type BuildingBlockLifecycleState string

var (
	BuildingBlockLifecycleStates                 = enum.Enum[BuildingBlockLifecycleState]{}
	BuildingBlockLifecycleStateActive            = BuildingBlockLifecycleStates.Entry("ACTIVE")
	BuildingBlockLifecycleStateMarkedForDeletion = BuildingBlockLifecycleStates.Entry("MARKED_FOR_DELETION")
	BuildingBlockLifecycleStateDeleted           = BuildingBlockLifecycleStates.Entry("DELETED")
)

type BuildingBlockStatus string

var (
	BuildingBlockStatuses                       = enum.Enum[BuildingBlockStatus]{}
	BuildingBlockStatusWaitingForDependentInput = BuildingBlockStatuses.Entry("WAITING_FOR_DEPENDENT_INPUT")
	BuildingBlockStatusWaitingForOperatorInput  = BuildingBlockStatuses.Entry("WAITING_FOR_OPERATOR_INPUT")
	BuildingBlockStatusWaitingForUserInput      = BuildingBlockStatuses.Entry("WAITING_FOR_USER_INPUT")
	BuildingBlockStatusWaitingForApproval       = BuildingBlockStatuses.Entry("WAITING_FOR_APPROVAL")
	BuildingBlockStatusPending                  = BuildingBlockStatuses.Entry("PENDING")
	BuildingBlockStatusInProgress               = BuildingBlockStatuses.Entry("IN_PROGRESS")
	BuildingBlockStatusSucceeded                = BuildingBlockStatuses.Entry("SUCCEEDED")
	BuildingBlockStatusFailed                   = BuildingBlockStatuses.Entry("FAILED")
	BuildingBlockStatusAborted                  = BuildingBlockStatuses.Entry("ABORTED")
)

type MeshBuildingBlockV2 struct {
	Metadata MeshBuildingBlockV2Metadata `json:"metadata" tfsdk:"metadata"`
	Spec     MeshBuildingBlockV2Spec     `json:"spec" tfsdk:"spec"`
	Status   *MeshBuildingBlockV2Status  `json:"status" tfsdk:"status"`
}

type MeshBuildingBlockV2Metadata struct {
	Uuid             *string `json:"uuid" tfsdk:"uuid"`
	OwnedByWorkspace string  `json:"ownedByWorkspace" tfsdk:"owned_by_workspace"`
}

type MeshBuildingBlockV2Spec struct {
	BuildingBlockDefinitionVersionRef MeshBuildingBlockV2DefinitionVersionRef `json:"buildingBlockDefinitionVersionRef" tfsdk:"building_block_definition_version_ref"`
	TargetRef                         MeshBuildingBlockV2TargetRef            `json:"targetRef" tfsdk:"target_ref"`
	DisplayName                       string                                  `json:"displayName" tfsdk:"display_name"`

	// Inputs as pointer MeshBuildingBlockInput to support mocking secret responses.
	Inputs               map[string]*MeshBuildingBlockInput `json:"inputs" tfsdk:"inputs"`
	ParentBuildingBlocks types.Set[MeshBuildingBlockParent] `json:"parentBuildingBlocks" tfsdk:"parent_building_blocks"`
}

type MeshBuildingBlockInput struct {
	Value          types.SecretOrAny                                `json:"value" tfsdk:"value"`
	ValueType      *enum.Entry[MeshBuildingBlockIOType]             `json:"valueType,omitempty" tfsdk:"-"`
	AssignmentType enum.Entry[MeshBuildingBlockInputAssignmentType] `json:"assignmentType,omitempty" tfsdk:"-"`

	// If IsSensitive is true, the [types.Variant] (typedef [types.SecretOrAny]) for Value field
	// is of [types.Secret] (case [types.Variant.X]).
	// Otherwise, the [types.Variant] is of [types.Any] (case [types.Variant.Y]).
	// As this is a fallback detection when JSON (un)marshaling,
	// types.Any must go second as [types.Variant] intentionally prefers X over Y.
	IsSensitive bool `json:"isSensitive" tfsdk:"-"`
}

func (m *MeshBuildingBlockInput) UnmarshalJSON(bytes []byte) error {
	type wrapped MeshBuildingBlockInput
	var target wrapped
	if err := json.Unmarshal(bytes, &target); err != nil {
		return err
	}
	*m = MeshBuildingBlockInput(target)
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
		moveXtoYIfPresent(&m.Value)
		return errors.Join(errs...)
	case m.Value.HasY():
		return fmt.Errorf("got sensitive argument or default_value but variant Y is set instead")
	default:
		return nil
	}
}

type MeshBuildingBlockV2DefinitionVersionRef struct {
	Uuid string `json:"uuid" tfsdk:"uuid"`
	// ContentHash is a Terraform-only field (json:"-", never sent to or returned by the backend).
	// It lets a config signal that the referenced version's content changed so a rerun is triggered
	// even though the version uuid is unchanged. The building_block (v3) resource honors it via the
	// shared rerunNeeded predicate used by both ModifyPlan and Update.
	ContentHash *string `json:"-" tfsdk:"content_hash"`
}

type MeshBuildingBlockV2TargetRef struct {
	Kind string  `json:"kind" tfsdk:"kind"`
	Uuid *string `json:"uuid" tfsdk:"uuid"`
	Name *string `json:"name" tfsdk:"name"`
}

type MeshBuildingBlockV2Lifecycle struct {
	State enum.Entry[BuildingBlockLifecycleState] `json:"state" tfsdk:"state"`
}

type MeshBuildingBlockV2Status struct {
	Status     enum.Entry[BuildingBlockStatus]    `json:"status" tfsdk:"status"`
	Outputs    map[string]MeshBuildingBlockOutput `json:"outputs" tfsdk:"outputs"`
	ForcePurge bool                               `json:"forcePurge" tfsdk:"force_purge"`
	Lifecycle  MeshBuildingBlockV2Lifecycle       `json:"lifecycle" tfsdk:"-"`
	// LatestRunUuid is nil if permissions don't allow reading the run (e.g. because run_transparency is false).
	// It tracks the latest *modifying* (apply/destroy) run and excludes dry runs.
	LatestRunUuid *string `json:"latestRunUuid" tfsdk:"latest_run_uuid"`
	// LatestDryRunUuid is the latest dry (DETECT) run, but only when it is the newest run; nil otherwise.
	// Same permission gating and nullability caveat as LatestRunUuid.
	LatestDryRunUuid *string `json:"latestDryRunUuid" tfsdk:"latest_dry_run_uuid"`
}

type MeshBuildingBlockOutput struct {
	Value          types.Any                                                   `json:"value" tfsdk:"value"`
	ValueType      enum.Entry[MeshBuildingBlockIOType]                         `json:"valueType" tfsdk:"value_type"`
	AssignmentType enum.Entry[MeshBuildingBlockDefinitionOutputAssignmentType] `json:"assignmentType" tfsdk:"assignment_type"`
}

// MeshBuildingBlockV2ListFilter holds the optional query filters for listing building blocks
// via the v2-preview list endpoint. All scalar fields are nil when unset (omitted from the
// query). The backend returns only active building blocks; soft-deleted ones are not listed.
type MeshBuildingBlockV2ListFilter struct {
	WorkspaceIdentifier *string
	ProjectIdentifier   *string
	PlatformIdentifier  *string
	Name                *string
	// DefinitionUuid filters by the owning building block definition's UUID (not a version).
	DefinitionUuid *string
	// VersionUuid filters by a specific building block definition version UUID.
	VersionUuid *string
	// VersionNumber filters by the literal definition version number. The backend parses it
	// leniently, so both "v1" and "1" match version 1.
	VersionNumber *string
	TenantUuid    *string
	// TargetKind filters by target ref kind, one of meshTenant or meshWorkspace.
	TargetKind *string
	Status     *string
	// ManagedByWorkspaceIdentifier and ManagedByDefinitionUuid select the platform-operator
	// (managed) permission scope: building blocks created from definitions owned by the given
	// workspace / definition. Requires the MANAGED_BUILDINGBLOCK_LIST authority.
	ManagedByWorkspaceIdentifier *string
	ManagedByDefinitionUuid      *string
}

type MeshBuildingBlockV2Client interface {
	Read(ctx context.Context, uuid string) (*MeshBuildingBlockV2, error)
	ReadFunc(uuid string) func(ctx context.Context) (*MeshBuildingBlockV2, error)
	List(ctx context.Context, filter *MeshBuildingBlockV2ListFilter) ([]MeshBuildingBlockV2, error)
	Create(ctx context.Context, bb *MeshBuildingBlockV2) (*MeshBuildingBlockV2, error)
	Update(ctx context.Context, bb *MeshBuildingBlockV2) (*MeshBuildingBlockV2, error)
	Delete(ctx context.Context, uuid string, purge bool) error
	TriggerRun(ctx context.Context, uuid string) error
}

type meshBuildingBlockV2Client struct {
	meshObject internal.MeshObjectClient[MeshBuildingBlockV2]
}

func newBuildingBlockV2Client(ctx context.Context, httpClient internal.HttpClient) MeshBuildingBlockV2Client {
	return meshBuildingBlockV2Client{internal.NewMeshObjectClient[MeshBuildingBlockV2](ctx, httpClient, "v2-preview")}
}

func (c meshBuildingBlockV2Client) Read(ctx context.Context, uuid string) (*MeshBuildingBlockV2, error) {
	return c.ReadFunc(uuid)(ctx)
}

func (c meshBuildingBlockV2Client) ReadFunc(uuid string) func(ctx context.Context) (*MeshBuildingBlockV2, error) {
	return func(ctx context.Context) (*MeshBuildingBlockV2, error) {
		return c.meshObject.Get(ctx, uuid)
	}
}

func (c meshBuildingBlockV2Client) List(ctx context.Context, filter *MeshBuildingBlockV2ListFilter) ([]MeshBuildingBlockV2, error) {
	var options []internal.RequestOption

	// Map each non-nil scalar filter to its query param. Names must match the backend
	// fetchBuildingBlocksV2 @RequestParam names exactly; a typo silently disables the filter.
	if filter != nil {
		for key, value := range map[string]*string{
			"workspaceIdentifier":          filter.WorkspaceIdentifier,
			"projectIdentifier":            filter.ProjectIdentifier,
			"platformIdentifier":           filter.PlatformIdentifier,
			"name":                         filter.Name,
			"definitionUuid":               filter.DefinitionUuid,
			"versionUuid":                  filter.VersionUuid,
			"versionNumber":                filter.VersionNumber,
			"tenantUuid":                   filter.TenantUuid,
			"targetRefKind":                filter.TargetKind,
			"status":                       filter.Status,
			"managedByWorkspaceIdentifier": filter.ManagedByWorkspaceIdentifier,
			"managedByDefinitionUuid":      filter.ManagedByDefinitionUuid,
		} {
			if value != nil {
				options = append(options, internal.WithUrlQuery(key, *value))
			}
		}
	}

	return c.meshObject.List(ctx, options...)
}

func (c meshBuildingBlockV2Client) Create(ctx context.Context, bb *MeshBuildingBlockV2) (*MeshBuildingBlockV2, error) {
	return c.meshObject.Post(ctx, bb)
}

func (c meshBuildingBlockV2Client) Update(ctx context.Context, bb *MeshBuildingBlockV2) (*MeshBuildingBlockV2, error) {
	if bb.Metadata.Uuid == nil {
		return nil, fmt.Errorf("cannot update building block without UUID")
	}
	return c.meshObject.Put(ctx, *bb.Metadata.Uuid, bb)
}

func (c meshBuildingBlockV2Client) Delete(ctx context.Context, uuid string, purge bool) error {
	var options []internal.RequestOption
	if purge {
		options = append(options, internal.WithPathElems("purge"))
	}
	return c.meshObject.Delete(ctx, uuid, options...)
}

// IsWaitingForInput reports whether the building block run is paused awaiting
// human input, a dependency, or an approval. Such a run will not progress on its
// own, so polling callers treat it as a terminal (but non-fatal) state and surface
// a warning.
func (bb *MeshBuildingBlockV2) IsWaitingForInput() bool {
	return bb.Status.Status == BuildingBlockStatusWaitingForOperatorInput ||
		bb.Status.Status == BuildingBlockStatusWaitingForUserInput ||
		bb.Status.Status == BuildingBlockStatusWaitingForDependentInput ||
		bb.Status.Status == BuildingBlockStatusWaitingForApproval
}

// bbUuidOrUnknown returns the building block UUID for diagnostic messages, or "<unknown>" if nil.
func bbUuidOrUnknown(bb *MeshBuildingBlockV2) string {
	if bb != nil && bb.Metadata.Uuid != nil {
		return *bb.Metadata.Uuid
	}
	return "<unknown>"
}

func (bb *MeshBuildingBlockV2) CreateSuccessful() (done bool, err error) {
	switch {
	case bb == nil:
		err = fmt.Errorf("building block not found after creation")
	case bb.Status == nil:
		// no status yet — keep polling
	case bb.Status.Status == BuildingBlockStatusFailed,
		bb.Status.Status == BuildingBlockStatusAborted:
		err = fmt.Errorf("building block %s reached %s state, check run logs in meshStack", bbUuidOrUnknown(bb), bb.Status.Status)
	case bb.IsWaitingForInput():
		// Paused awaiting input — stop polling so the caller can surface a warning.
		done = true
	case bb.Status.Status == BuildingBlockStatusSucceeded:
		done = true
	case !slices.Contains(BuildingBlockStatuses, bb.Status.Status):
		// Unrecognized status: fail fast instead of polling to the timeout — the backend returned a
		// status this provider version does not know about (provider may be out of date).
		err = fmt.Errorf("unknown building block status %q for building block %s; provider may be out of date", bb.Status.Status, bbUuidOrUnknown(bb))
	}
	return
}

func (bb *MeshBuildingBlockV2) DeletionSuccessful() (done bool, err error) {
	switch {
	case bb == nil:
		// 404: the block was hard-removed (e.g. its definition was deleted too); treat as done.
		done = true
	case bb.Status != nil && bb.Status.Lifecycle.State == BuildingBlockLifecycleStateDeleted:
		// Soft delete: once deletion completes the backend keeps returning the block with lifecycle
		// DELETED (it does not 404), so treat DELETED as done. While deletion is still in progress the
		// block is returned with MARKED_FOR_DELETION, which falls through as not-yet-done so we keep polling.
		done = true
	case bb.Status != nil && bb.Status.Status == BuildingBlockStatusFailed:
		// A force-purge (definition deletion_mode = PURGE, or an admin purge) deletes the block
		// regardless of its delete run's outcome, so a FAILED status here is transient — the
		// lifecycle still proceeds to DELETED. Keep polling instead of erroring on that transient
		// FAILED. Only a FAILED delete that is NOT being force-purged is a genuine stuck deletion.
		if !bb.Status.ForcePurge {
			err = fmt.Errorf("building block %s reached FAILED state during deletion. For more details, check the building block run logs in meshStack", bbUuidOrUnknown(bb))
		}
	}
	return
}

func (c meshBuildingBlockV2Client) TriggerRun(ctx context.Context, bbUuid string) error {
	// trigger-run returns an empty 2xx body; use DoAuthorizedRequest[any] to signal no body expected.
	// No body is sent, so the backend triggers a normal (non-dry) apply run.
	_, err := internal.DoAuthorizedRequest[any](
		ctx,
		c.meshObject.HttpClient,
		"POST",
		c.meshObject.ApiUrl.JoinPath(bbUuid, "trigger-run"),
		internal.WithAccept(c.meshObject.MeshObjectMimeType()),
	)
	return err
}
