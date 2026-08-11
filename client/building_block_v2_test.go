package client

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/meshcloud/terraform-provider-meshstack/client/types/enum"
)

const (
	testParentUuid           = "11111111-1111-1111-1111-111111111111"
	testParentDefinitionUuid = "22222222-2222-2222-2222-222222222222"
	// testParentRef is the shape this provider sends for a parent in parentBuildingBlockRefs.
	testParentRef = `{"kind": "meshBuildingBlock", "uuid": "` + testParentUuid + `"}`
	// testDeprecatedParent is what the provider sends alongside it, for a backend that does not know
	// parentBuildingBlockRefs yet.
	testDeprecatedParent = `{"buildingBlockUuid": "` + testParentUuid + `"}`
)

func TestMeshBuildingBlockV2_DeletionSuccessful(t *testing.T) {
	tests := []struct {
		name     string
		bb       *MeshBuildingBlockV2
		wantDone bool
		wantErr  bool
	}{
		{
			name:     "nil (404 — hard deletion / purge)",
			bb:       nil,
			wantDone: true,
			wantErr:  false,
		},
		{
			name: "lifecycle state DELETED (soft delete completed, block still returned)",
			bb: &MeshBuildingBlockV2{
				Status: &MeshBuildingBlockV2Status{
					Lifecycle: MeshBuildingBlockV2Lifecycle{State: BuildingBlockLifecycleStateDeleted},
				},
			},
			wantDone: true,
			wantErr:  false,
		},
		{
			name: "status FAILED during deletion",
			bb: &MeshBuildingBlockV2{
				Metadata: MeshBuildingBlockV2Metadata{Uuid: new("test-uuid")},
				Status: &MeshBuildingBlockV2Status{
					Status: BuildingBlockStatusFailed,
				},
			},
			wantDone: false,
			wantErr:  true,
		},
		{
			name: "status FAILED but force-purged keeps polling (transient, will reach DELETED)",
			bb: &MeshBuildingBlockV2{
				Metadata: MeshBuildingBlockV2Metadata{Uuid: new("test-uuid")},
				Status: &MeshBuildingBlockV2Status{
					Status:     BuildingBlockStatusFailed,
					ForcePurge: true,
				},
			},
			wantDone: false,
			wantErr:  false,
		},
		{
			name: "status FAILED with nil Uuid does not panic",
			bb: &MeshBuildingBlockV2{
				Metadata: MeshBuildingBlockV2Metadata{Uuid: nil},
				Status: &MeshBuildingBlockV2Status{
					Status: BuildingBlockStatusFailed,
				},
			},
			wantDone: false,
			wantErr:  true,
		},
		{
			name: "still in progress (MARKED_FOR_DELETION lifecycle, non-failed status)",
			bb: &MeshBuildingBlockV2{
				Status: &MeshBuildingBlockV2Status{
					Lifecycle: MeshBuildingBlockV2Lifecycle{State: BuildingBlockLifecycleStateMarkedForDeletion},
				},
			},
			wantDone: false,
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			done, err := tt.bb.DeletionSuccessful()
			assert.Equal(t, tt.wantDone, done)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestMeshBuildingBlockV2_CreateSuccessful(t *testing.T) {
	tests := []struct {
		name        string
		bb          *MeshBuildingBlockV2
		wantDone    bool
		wantErr     bool
		errContains string
	}{
		{
			name:     "nil (not found after creation)",
			bb:       nil,
			wantDone: false,
			wantErr:  true,
		},
		{
			name:     "no status yet — keep polling",
			bb:       &MeshBuildingBlockV2{Metadata: MeshBuildingBlockV2Metadata{Uuid: new("test-uuid")}},
			wantDone: false,
			wantErr:  false,
		},
		{
			name: "SUCCEEDED",
			bb: &MeshBuildingBlockV2{
				Metadata: MeshBuildingBlockV2Metadata{Uuid: new("test-uuid")},
				Status:   &MeshBuildingBlockV2Status{Status: BuildingBlockStatusSucceeded},
			},
			wantDone: true,
			wantErr:  false,
		},
		{
			name: "FAILED",
			bb: &MeshBuildingBlockV2{
				Metadata: MeshBuildingBlockV2Metadata{Uuid: new("test-uuid")},
				Status:   &MeshBuildingBlockV2Status{Status: BuildingBlockStatusFailed},
			},
			wantDone: false,
			wantErr:  true,
		},
		{
			name: "ABORTED",
			bb: &MeshBuildingBlockV2{
				Metadata: MeshBuildingBlockV2Metadata{Uuid: new("test-uuid")},
				Status:   &MeshBuildingBlockV2Status{Status: BuildingBlockStatusAborted},
			},
			wantDone: false,
			wantErr:  true,
		},
		{
			name: "WAITING_FOR_USER_INPUT — terminal but non-fatal",
			bb: &MeshBuildingBlockV2{
				Metadata: MeshBuildingBlockV2Metadata{Uuid: new("test-uuid")},
				Status:   &MeshBuildingBlockV2Status{Status: BuildingBlockStatusWaitingForUserInput},
			},
			wantDone: true,
			wantErr:  false,
		},
		{
			name: "WAITING_FOR_APPROVAL — terminal but non-fatal",
			bb: &MeshBuildingBlockV2{
				Metadata: MeshBuildingBlockV2Metadata{Uuid: new("test-uuid")},
				Status:   &MeshBuildingBlockV2Status{Status: BuildingBlockStatusWaitingForApproval},
			},
			wantDone: true,
			wantErr:  false,
		},
		{
			name: "FAILED with nil Uuid does not panic",
			bb: &MeshBuildingBlockV2{
				Metadata: MeshBuildingBlockV2Metadata{Uuid: nil},
				Status:   &MeshBuildingBlockV2Status{Status: BuildingBlockStatusFailed},
			},
			wantDone:    false,
			wantErr:     true,
			errContains: "<unknown>",
		},
		{
			name: "unknown status — fail fast",
			bb: &MeshBuildingBlockV2{
				Metadata: MeshBuildingBlockV2Metadata{Uuid: new("test-uuid")},
				Status:   &MeshBuildingBlockV2Status{Status: enum.Entry[BuildingBlockStatus]("SOMETHING_NEW")},
			},
			wantDone:    false,
			wantErr:     true,
			errContains: "unknown building block status",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			done, err := tt.bb.CreateSuccessful()
			assert.Equal(t, tt.wantDone, done)
			if tt.wantErr {
				require.Error(t, err)
				if tt.errContains != "" {
					assert.Contains(t, err.Error(), tt.errContains)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// TestMeshBuildingBlockV2Parent_UnmarshalJSON covers every response shape. Terraform has to see the
// same {kind, uuid} against every backend, so that set hashing and UseStateForUnknown stay stable.
func TestMeshBuildingBlockV2Parent_UnmarshalJSON(t *testing.T) {
	tests := []struct {
		name               string
		response           string
		wantDefinitionUuid string
	}{
		{
			// An older backend reports the parent inside a buildingBlockRef envelope, which this provider
			// does not read, so only the deprecated field carries the uuid.
			name: "enveloped response without a top-level uuid",
			response: `{
			  "buildingBlockRef": {"kind": "meshBuildingBlock", "uuid": "` + testParentUuid + `"},
			  "buildingBlockUuid": "` + testParentUuid + `",
			  "definitionUuid": "` + testParentDefinitionUuid + `"
			}`,
			wantDefinitionUuid: testParentDefinitionUuid,
		},
		{
			name: "flattened response that also carries a top-level uuid",
			response: `{
			  "kind": "meshBuildingBlock",
			  "uuid": "` + testParentUuid + `",
			  "buildingBlockUuid": "` + testParentUuid + `",
			  "definitionUuid": "` + testParentDefinitionUuid + `"
			}`,
			wantDefinitionUuid: testParentDefinitionUuid,
		},
		{
			name:     "flattened response once the deprecated fields are gone",
			response: `{"kind": "meshBuildingBlock", "uuid": "` + testParentUuid + `"}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var parent MeshBuildingBlockV2Parent
			require.NoError(t, json.Unmarshal([]byte(tt.response), &parent))
			assert.Equal(t, MeshBuildingBlockV2Parent{
				UuidRef:           UuidRef{Kind: MeshObjectKind.BuildingBlock, Uuid: testParentUuid},
				BuildingBlockUuid: testParentUuid,
				DefinitionUuid:    tt.wantDefinitionUuid,
			}, parent)
		})
	}
}

// TestMeshBuildingBlockV2Spec_ParentsRoundTrip goes through the whole spec, so a change to the json
// tag of parentBuildingBlockRefs or to the Set element type is caught too.
func TestMeshBuildingBlockV2Spec_ParentsRoundTrip(t *testing.T) {
	const response = `{
	  "buildingBlockDefinitionVersionRef": {"kind": "meshBuildingBlockDefinitionVersion", "uuid": "33333333-3333-3333-3333-333333333333"},
	  "targetRef": {"kind": "meshWorkspace", "name": "my-workspace"},
	  "displayName": "child",
	  "inputs": {},
	  "parentBuildingBlockRefs": [` + testParentRef + `],
	  "parentBuildingBlocks": [{"buildingBlockUuid": "` + testParentUuid + `", "definitionUuid": "` + testParentDefinitionUuid + `"}]
	}`

	var spec MeshBuildingBlockV2Spec
	require.NoError(t, json.Unmarshal([]byte(response), &spec))
	require.Len(t, spec.ParentBuildingBlockRefs, 1)
	assert.Equal(t, UuidRef{Kind: MeshObjectKind.BuildingBlock, Uuid: testParentUuid}, spec.ParentBuildingBlockRefs[0])
	require.Len(t, spec.ParentBuildingBlocks, 1)
	assert.Equal(t, testParentDefinitionUuid, spec.ParentBuildingBlocks[0].DefinitionUuid)

	assertSentUnderBothFieldNames(t, spec)
}

// TestMeshBuildingBlockV2Spec_ParentsFromDeprecatedFieldOnly covers a backend that does not serve
// parentBuildingBlockRefs yet, and the deprecated meshstack_building_block_v2 surfaces, which fill
// only the deprecated field.
func TestMeshBuildingBlockV2Spec_ParentsFromDeprecatedFieldOnly(t *testing.T) {
	const response = `{
	  "buildingBlockDefinitionVersionRef": {"kind": "meshBuildingBlockDefinitionVersion", "uuid": "33333333-3333-3333-3333-333333333333"},
	  "targetRef": {"kind": "meshWorkspace", "name": "my-workspace"},
	  "displayName": "child",
	  "inputs": {},
	  "parentBuildingBlocks": [{"buildingBlockUuid": "` + testParentUuid + `", "definitionUuid": "` + testParentDefinitionUuid + `"}]
	}`

	var spec MeshBuildingBlockV2Spec
	require.NoError(t, json.Unmarshal([]byte(response), &spec))
	require.Len(t, spec.ParentBuildingBlockRefs, 1)
	assert.Equal(t, UuidRef{Kind: MeshObjectKind.BuildingBlock, Uuid: testParentUuid}, spec.ParentBuildingBlockRefs[0])

	assertSentUnderBothFieldNames(t, spec)
}

func assertSentUnderBothFieldNames(t *testing.T, spec MeshBuildingBlockV2Spec) {
	t.Helper()

	out, err := json.Marshal(spec)
	require.NoError(t, err)
	var request map[string]json.RawMessage
	require.NoError(t, json.Unmarshal(out, &request))
	assert.JSONEq(t, "["+testParentRef+"]", string(request["parentBuildingBlockRefs"]))
	assert.JSONEq(t, "["+testDeprecatedParent+"]", string(request["parentBuildingBlocks"]))
}
