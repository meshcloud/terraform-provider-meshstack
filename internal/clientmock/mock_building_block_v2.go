package clientmock

import (
	"context"
	"encoding/json/v2"
	"fmt"
	"slices"

	"github.com/google/uuid"
	"github.com/meshcloud/meshstack-cli/client"
	clientTypes "github.com/meshcloud/meshstack-cli/client/types"
)

// wireCompatibility repeats what internal/provider marshals with, so that the round-trip below
// keeps a nil collection nil instead of handing the test an empty one.
var wireCompatibility = json.JoinOptions(
	json.Deterministic(true),
	json.FormatNilSliceAsNull(true),
	json.FormatNilMapAsNull(true),
)

// deepCopyBB returns a deep copy of bb via JSON round-trip, isolating the store from caller mutations.
func deepCopyBB(bb *client.MeshBuildingBlockV2) *client.MeshBuildingBlockV2 {
	if bb == nil {
		return nil
	}
	data, err := json.Marshal(bb, wireCompatibility)
	if err != nil {
		panic(fmt.Sprintf("deepCopyBB: marshal failed: %v", err))
	}
	var cp client.MeshBuildingBlockV2
	if err := json.Unmarshal(data, &cp); err != nil {
		panic(fmt.Sprintf("deepCopyBB: unmarshal failed: %v", err))
	}
	return &cp
}

// materializeNullRows adds null-valued USER_INPUT rows for every definition input that is not
// already present in the building block's inputs. This mirrors the real meshStack backend
// which returns a {assignmentType: USER_INPUT, value: null} entry for every definition input
// the request didn't supply, so that provider tests see the same shape as the real backend.
func materializeNullRows(inputs map[string]*client.MeshBuildingBlockInput, bbdVersionStore *Store[client.MeshBuildingBlockDefinitionVersion], versionRefUuid string) {
	if bbdVersionStore == nil || versionRefUuid == "" {
		return
	}
	version, ok := bbdVersionStore.Get(versionRefUuid)
	if !ok {
		return
	}
	for key, defInput := range version.Spec.Inputs {
		if _, exists := inputs[key]; exists {
			continue
		}
		if defInput.AssignmentType != client.MeshBuildingBlockInputAssignmentTypeUserInput.Unwrap() {
			// Only materialize USER_INPUT rows; operator/static inputs are not echoed back as null
			continue
		}
		// Insert a null-valued USER_INPUT row.
		inputs[key] = &client.MeshBuildingBlockInput{
			Value:          clientTypes.SecretOrAny{},
			AssignmentType: client.MeshBuildingBlockInputAssignmentTypeUserInput,
		}
	}
}

type MeshBuildingBlockV2Client struct {
	Store           *Store[client.MeshBuildingBlockV2]
	BbdVersionStore *Store[client.MeshBuildingBlockDefinitionVersion]
}

// withDerivedParents rebuilds the deprecated parent entries, which do not survive deepCopyBB because
// the wire format carries no definition uuid. The backend derives the same value from the referenced
// building block for every response, and status.basedOnDraftVersion from the block's own version.
func (m MeshBuildingBlockV2Client) withDerivedParents(bb *client.MeshBuildingBlockV2) *client.MeshBuildingBlockV2 {
	if bb == nil {
		return nil
	}
	parents := make(clientTypes.Set[client.MeshBuildingBlockV2Parent], 0, len(bb.Spec.ParentBuildingBlockRefs))
	for _, ref := range bb.Spec.ParentBuildingBlockRefs {
		parent := client.MeshBuildingBlockV2Parent{
			UuidRef:           client.UuidRef{Kind: client.MeshObjectKind.BuildingBlock, Uuid: ref.Uuid},
			BuildingBlockUuid: ref.Uuid,
		}
		if storedParent, ok := m.Store.Get(ref.Uuid); ok {
			version, ok := m.BbdVersionStore.Get(storedParent.Spec.BuildingBlockDefinitionVersionRef.Uuid)
			if ok && version.Spec.BuildingBlockDefinitionRef != nil {
				parent.DefinitionUuid = version.Spec.BuildingBlockDefinitionRef.Uuid
			}
		}
		parents = append(parents, parent)
	}
	bb.Spec.ParentBuildingBlocks = parents
	if bb.Status != nil {
		version, ok := m.BbdVersionStore.Get(bb.Spec.BuildingBlockDefinitionVersionRef.Uuid)
		bb.Status.BasedOnDraftVersion = ok && version.Spec.State != nil && *version.Spec.State == client.MeshBuildingBlockDefinitionVersionStateDraft.Unwrap()
	}
	return bb
}

func (m MeshBuildingBlockV2Client) Read(_ context.Context, bbUuid string) (*client.MeshBuildingBlockV2, error) {
	if bb, ok := m.Store.Get(bbUuid); ok {
		return m.withDerivedParents(deepCopyBB(bb)), nil
	}
	return nil, nil
}

func (m MeshBuildingBlockV2Client) ReadFunc(bbUuid string) func(ctx context.Context) (*client.MeshBuildingBlockV2, error) {
	return func(ctx context.Context) (*client.MeshBuildingBlockV2, error) {
		return m.Read(ctx, bbUuid)
	}
}

func (m MeshBuildingBlockV2Client) List(_ context.Context, filter client.MeshBuildingBlockV2ListFilter) ([]client.MeshBuildingBlockV2, error) {
	result := make([]client.MeshBuildingBlockV2, 0)
	// Iterate in sorted-uuid order for deterministic test output.
	for _, key := range m.Store.SortedKeys() {
		bb, ok := m.Store.Get(key)
		if !ok {
			continue
		}
		if !mockBuildingBlockMatchesFilter(bb, filter) {
			continue
		}
		result = append(result, *m.withDerivedParents(deepCopyBB(bb)))
	}
	return result, nil
}

// mockBuildingBlockMatchesFilter applies the subset of MeshBuildingBlockV2ListFilter fields that
// are derivable from a stored building block. Fields the mock store doesn't carry — DefinitionUuid
// and VersionNumber (the store only holds the definition *version* uuid, not the definition uuid or
// the version number) — are accepted but not applied, so tests should assert only on the supported
// filters. The real backend applies all of them.
func mockBuildingBlockMatchesFilter(bb *client.MeshBuildingBlockV2, filter client.MeshBuildingBlockV2ListFilter) bool {
	if filter.WorkspaceIdentifier != nil && bb.Metadata.OwnedByWorkspace != *filter.WorkspaceIdentifier {
		return false
	}
	if filter.Name != nil && bb.Spec.DisplayName != *filter.Name {
		return false
	}
	if filter.VersionUuid != nil && bb.Spec.BuildingBlockDefinitionVersionRef.Uuid != *filter.VersionUuid {
		return false
	}
	if filter.TargetKind != nil && bb.Spec.TargetRef.Kind != *filter.TargetKind {
		return false
	}
	if filter.TenantUuid != nil && (bb.Spec.TargetRef.Uuid == nil || *bb.Spec.TargetRef.Uuid != *filter.TenantUuid) {
		return false
	}
	if filter.Status != nil && (bb.Status == nil || string(bb.Status.Status) != *filter.Status) {
		return false
	}
	if filter.HealthStatus != nil && (bb.Status == nil || !slices.Contains(filter.HealthStatus, bb.Status.HealthStatus.String())) {
		return false
	}
	if filter.Action != nil && (bb.Status == nil || !slices.Contains(filter.Action, bb.Status.Action.String())) {
		return false
	}
	return true
}

func (m MeshBuildingBlockV2Client) Create(_ context.Context, bb *client.MeshBuildingBlockV2) (*client.MeshBuildingBlockV2, error) {
	id := uuid.NewString()

	ownedByWorkspace := ""
	if bb.Spec.TargetRef.Name != nil {
		ownedByWorkspace = *bb.Spec.TargetRef.Name
	}

	// Deep-copy the incoming DTO before any mutation so we never modify the caller's data.
	stored := deepCopyBB(bb)

	for _, input := range stored.Spec.Inputs {
		if input.AssignmentType == "" {
			input.AssignmentType = client.MeshBuildingBlockInputAssignmentTypeUserInput
		}
	}
	// Inputs are pointer-valued so the shared backendSecretBehavior walker can reach (and mutate) the
	// secret inside each addressable map value. On create there is no prior block to validate against.
	backendSecretBehavior(true, stored, nil)

	// Materialize null-valued USER_INPUT rows for unconfigured definition inputs,
	// mirroring real backend behaviour.
	materializeNullRows(stored.Spec.Inputs, m.BbdVersionStore, stored.Spec.BuildingBlockDefinitionVersionRef.Uuid)

	stored.Metadata = client.MeshBuildingBlockV2Metadata{
		Uuid:             &id,
		OwnedByWorkspace: ownedByWorkspace,
	}
	stored.Status = finishedRun(client.MeshBuildingBlockV2Status{
		Lifecycle: client.MeshBuildingBlockV2Lifecycle{State: client.BuildingBlockLifecycleStateActive},
	})

	m.Store.Set(id, stored)
	// Return a fresh deep copy so SetFromClientDto cannot mutate the store via the returned pointer.
	return m.withDerivedParents(deepCopyBB(stored)), nil
}

func (m MeshBuildingBlockV2Client) Update(_ context.Context, bb *client.MeshBuildingBlockV2) (*client.MeshBuildingBlockV2, error) {
	if bb.Metadata.Uuid == nil {
		return nil, fmt.Errorf("cannot update building block without UUID")
	}

	stored := deepCopyBB(bb) // deep-copy in (see Create)

	for _, input := range stored.Spec.Inputs {
		if input.AssignmentType == "" {
			input.AssignmentType = client.MeshBuildingBlockInputAssignmentTypeUserInput
		}
	}

	// Validate/hash secrets against the stored block: an unchanged secret is sent hash-only and must
	// match the stored hash, while a rotated secret arrives as plaintext and is re-hashed. Pointer-valued
	// inputs let the shared walker reach each addressable secret directly.
	prior, ok := m.Store.Get(*stored.Metadata.Uuid)
	if !ok {
		return nil, fmt.Errorf("building block %q not found", *stored.Metadata.Uuid)
	}
	existing := deepCopyBB(prior)

	// Mirror the backend: a PUT patches only the inputs it carries, so inputs omitted from the request
	// keep their existing values. Without this the mock would null out the other party's inputs when a
	// configuration manages only a subset (e.g. a platform operator sending only operator inputs).
	if stored.Spec.Inputs == nil {
		stored.Spec.Inputs = map[string]*client.MeshBuildingBlockInput{}
	}
	for key, prior := range existing.Spec.Inputs {
		if _, sent := stored.Spec.Inputs[key]; !sent {
			stored.Spec.Inputs[key] = prior
		}
	}

	backendSecretBehavior(false, stored, existing)

	materializeNullRows(stored.Spec.Inputs, m.BbdVersionStore, stored.Spec.BuildingBlockDefinitionVersionRef.Uuid) // see Create

	// The provider waits for a run that differs from the answer, so the answer keeps the run before.
	answered := deepCopyBB(stored)
	answered.Status = existing.Status
	stored.Status = finishedRun(*existing.Status)

	m.Store.Set(*stored.Metadata.Uuid, stored)
	return m.withDerivedParents(answered), nil
}

func (m MeshBuildingBlockV2Client) Delete(_ context.Context, bbUuid string, purge bool) error {
	m.Store.Delete(bbUuid)
	return nil
}

func (m MeshBuildingBlockV2Client) TriggerRun(_ context.Context, bbUuid string) (*client.MeshBuildingBlockV2, error) {
	bb, ok := m.Store.Get(bbUuid)
	if !ok {
		return nil, fmt.Errorf("building block %q not found", bbUuid)
	}
	stored := deepCopyBB(bb)
	stored.Status = finishedRun(*bb.Status)
	m.Store.Set(bbUuid, stored)
	// The answer keeps the run before, see Update.
	return m.withDerivedParents(deepCopyBB(bb)), nil
}

func finishedRun(status client.MeshBuildingBlockV2Status) *client.MeshBuildingBlockV2Status {
	status.Status = client.BuildingBlockStatusSucceeded
	status.HealthStatus = client.BuildingBlockHealthStatusSucceeded
	status.Action = client.BuildingBlockActionNoAction
	status.LatestRunUuid = new(uuid.NewString())
	status.LatestDryRunUuid = nil
	return &status
}
