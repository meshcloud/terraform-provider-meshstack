package clientmock

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"

	"github.com/google/uuid"

	"github.com/meshcloud/terraform-provider-meshstack/client"
	clientTypes "github.com/meshcloud/terraform-provider-meshstack/client/types"
)

// deepCopyBB returns a deep copy of bb via JSON round-trip.
// This ensures that mutations to the returned value do not affect the stored state,
// and mutations to the caller's value do not affect the stored state.
func deepCopyBB(bb *client.MeshBuildingBlockV2) *client.MeshBuildingBlockV2 {
	if bb == nil {
		return nil
	}
	data, err := json.Marshal(bb)
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

func (m MeshBuildingBlockV2Client) Read(_ context.Context, bbUuid string) (*client.MeshBuildingBlockV2, error) {
	if bb, ok := m.Store.Get(bbUuid); ok {
		return deepCopyBB(bb), nil
	}
	return nil, nil
}

func (m MeshBuildingBlockV2Client) ReadFunc(bbUuid string) func(ctx context.Context) (*client.MeshBuildingBlockV2, error) {
	return func(ctx context.Context) (*client.MeshBuildingBlockV2, error) {
		return m.Read(ctx, bbUuid)
	}
}

func (m MeshBuildingBlockV2Client) List(_ context.Context, filter *client.MeshBuildingBlockV2ListFilter) ([]client.MeshBuildingBlockV2, error) {
	result := make([]client.MeshBuildingBlockV2, 0)
	// Iterate in sorted-uuid order for deterministic test output.
	for _, key := range m.Store.SortedKeys() {
		bb, ok := m.Store.Get(key)
		if !ok {
			continue
		}
		if filter != nil && !mockBuildingBlockMatchesFilter(bb, filter) {
			continue
		}
		result = append(result, *deepCopyBB(bb))
	}
	return result, nil
}

// mockBuildingBlockMatchesFilter applies the subset of MeshBuildingBlockV2ListFilter fields that
// are derivable from a stored building block. Fields the mock store doesn't carry — DefinitionUuid
// and VersionNumber (the store only holds the definition *version* uuid, not the definition uuid or
// the version number) — are accepted but not applied, so tests should assert only on the supported
// filters. The real backend applies all of them.
func mockBuildingBlockMatchesFilter(bb *client.MeshBuildingBlockV2, filter *client.MeshBuildingBlockV2ListFilter) bool {
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
	return true
}

func (m MeshBuildingBlockV2Client) Create(_ context.Context, bb *client.MeshBuildingBlockV2) (*client.MeshBuildingBlockV2, error) {
	id := uuid.NewString()
	runUuid := uuid.NewString()

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
	stored.Status = &client.MeshBuildingBlockV2Status{
		Status:        client.BuildingBlockStatusSucceeded,
		LatestRunUuid: &runUuid,
		Lifecycle: client.MeshBuildingBlockV2Lifecycle{
			State: client.BuildingBlockLifecycleStateActive,
		},
	}

	m.Store.Set(id, stored)
	// Return a fresh deep copy so SetFromClientDto cannot mutate the store via the returned pointer.
	return deepCopyBB(stored), nil
}

func (m MeshBuildingBlockV2Client) Update(_ context.Context, bb *client.MeshBuildingBlockV2) (*client.MeshBuildingBlockV2, error) {
	if bb.Metadata.Uuid == nil {
		return nil, fmt.Errorf("cannot update building block without UUID")
	}

	// Deep-copy the incoming DTO before any mutation so we never modify the caller's data.
	stored := deepCopyBB(bb)

	for _, input := range stored.Spec.Inputs {
		if input.AssignmentType == "" {
			input.AssignmentType = client.MeshBuildingBlockInputAssignmentTypeUserInput
		}
	}

	// Validate/hash secrets against the stored block: an unchanged secret is sent hash-only and must
	// match the stored hash, while a rotated secret arrives as plaintext and is re-hashed. Pointer-valued
	// inputs let the shared walker reach each addressable secret directly.
	var existing *client.MeshBuildingBlockV2
	if prior, ok := m.Store.Get(*stored.Metadata.Uuid); ok {
		existing = deepCopyBB(prior)
	}

	// Mirror the backend: a PUT patches only the inputs it carries, so inputs omitted from the request
	// keep their existing values. Without this the mock would null out the other party's inputs when a
	// configuration manages only a subset (e.g. a platform operator sending only operator inputs).
	if existing != nil {
		if stored.Spec.Inputs == nil {
			stored.Spec.Inputs = map[string]*client.MeshBuildingBlockInput{}
		}
		for key, prior := range existing.Spec.Inputs {
			if _, sent := stored.Spec.Inputs[key]; !sent {
				stored.Spec.Inputs[key] = prior
			}
		}
	}

	backendSecretBehavior(false, stored, existing)

	// Materialize null-valued USER_INPUT rows for unconfigured definition inputs,
	// mirroring real backend behaviour.
	materializeNullRows(stored.Spec.Inputs, m.BbdVersionStore, stored.Spec.BuildingBlockDefinitionVersionRef.Uuid)

	if stored.Status == nil {
		// A PUT carries only Metadata+Spec (Status is read-only and not sent by the provider).
		// Mirror the real backend: the PUT triggers an apply run only on a real provisioning change
		// (version upgrade or an actual input/parent change); a rename / no-op carries the prior status
		// unchanged. (A content_hash-only change PUTs an identical spec → no run here; the provider forces
		// it via a separate TriggerRun.)
		switch {
		case existing == nil || existing.Status == nil:
			runUuid := uuid.NewString()
			stored.Status = &client.MeshBuildingBlockV2Status{
				Status:        client.BuildingBlockStatusSucceeded,
				LatestRunUuid: &runUuid,
				Lifecycle: client.MeshBuildingBlockV2Lifecycle{
					State: client.BuildingBlockLifecycleStateActive,
				},
			}
		case provisioningChanged(existing, stored):
			status := *existing.Status
			runUuid := uuid.NewString()
			status.Status = client.BuildingBlockStatusSucceeded
			status.LatestRunUuid = &runUuid
			status.LatestDryRunUuid = nil
			stored.Status = &status
		default:
			stored.Status = existing.Status
		}
	}

	m.Store.Set(*stored.Metadata.Uuid, stored)
	// Return a fresh deep copy so SetFromClientDto cannot mutate the store via the returned pointer.
	return deepCopyBB(stored), nil
}

func (m MeshBuildingBlockV2Client) Delete(_ context.Context, bbUuid string, purge bool) error {
	m.Store.Delete(bbUuid)
	return nil
}

func (m MeshBuildingBlockV2Client) TriggerRun(_ context.Context, bbUuid string) error {
	bb, ok := m.Store.Get(bbUuid)
	if !ok {
		return fmt.Errorf("building block %q not found", bbUuid)
	}
	cp := deepCopyBB(bb)
	if cp.Status == nil {
		cp.Status = &client.MeshBuildingBlockV2Status{
			Lifecycle: client.MeshBuildingBlockV2Lifecycle{
				State: client.BuildingBlockLifecycleStateActive,
			},
		}
	}
	cp.Status.Status = client.BuildingBlockStatusSucceeded
	cp.Status.LatestRunUuid = new(uuid.NewString())
	cp.Status.LatestDryRunUuid = nil
	m.Store.Set(bbUuid, cp)
	return nil
}

// provisioningChanged reports whether a PUT made a backend-visible change that triggers an apply run:
// a definition-version upgrade, an actual input change, or a parent change. A displayName-only rename or
// an identical re-PUT returns false (no run), mirroring the backend.
func provisioningChanged(existing, stored *client.MeshBuildingBlockV2) bool {
	if existing.Spec.BuildingBlockDefinitionVersionRef.Uuid != stored.Spec.BuildingBlockDefinitionVersionRef.Uuid {
		return true
	}
	if !reflect.DeepEqual(existing.Spec.Inputs, stored.Spec.Inputs) {
		return true
	}
	return !reflect.DeepEqual(existing.Spec.ParentBuildingBlocks, stored.Spec.ParentBuildingBlocks)
}
