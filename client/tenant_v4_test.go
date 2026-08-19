package client

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMeshTenant_DeletionSuccessful(t *testing.T) {
	tests := []struct {
		name     string
		tenant   *MeshTenant
		wantDone bool
	}{
		{
			name:     "nil (404 — tenant purged)",
			tenant:   nil,
			wantDone: true,
		},
		{
			name: "lifecycle DELETED (deletion completed, tenant still returned)",
			tenant: &MeshTenant{Status: MeshTenantStatus{
				Lifecycle: MeshTenantLifecycle{State: TenantLifecycleStateDeleted},
			}},
			wantDone: true,
		},
		{
			name: "lifecycle MARKED_FOR_DELETION (deletion still running)",
			tenant: &MeshTenant{Status: MeshTenantStatus{
				Lifecycle: MeshTenantLifecycle{
					State:             TenantLifecycleStateMarkedForDeletion,
					MarkedForDeletion: &MeshTenantLifecycleAction{Timestamp: "2026-07-30T16:14:14Z"},
				},
			}},
			wantDone: false,
		},
		{
			name: "lifecycle ACTIVE",
			tenant: &MeshTenant{Status: MeshTenantStatus{
				Lifecycle: MeshTenantLifecycle{State: TenantLifecycleStateActive},
			}},
			wantDone: false,
		},
		{
			name:     "no lifecycle reported",
			tenant:   &MeshTenant{Metadata: MeshTenantMetadata{Uuid: "test-uuid"}},
			wantDone: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			done, err := tt.tenant.DeletionSuccessful()
			assert.Equal(t, tt.wantDone, done)
			assert.NoError(t, err)
		})
	}
}

func TestMeshTenantV4_DeletionSuccessful(t *testing.T) {
	tests := []struct {
		name     string
		tenant   *MeshTenantV4
		wantDone bool
	}{
		{
			name:     "nil (404 — tenant purged)",
			tenant:   nil,
			wantDone: true,
		},
		{
			name: "lifecycle DELETED (deletion completed, tenant still returned)",
			tenant: &MeshTenantV4{Status: MeshTenantV4Status{
				Lifecycle: MeshTenantLifecycle{State: TenantLifecycleStateDeleted},
			}},
			wantDone: true,
		},
		{
			name: "lifecycle MARKED_FOR_DELETION (deletion still running)",
			tenant: &MeshTenantV4{Status: MeshTenantV4Status{
				Lifecycle: MeshTenantLifecycle{State: TenantLifecycleStateMarkedForDeletion},
			}},
			wantDone: false,
		},
		{
			name: "lifecycle ACTIVE",
			tenant: &MeshTenantV4{Status: MeshTenantV4Status{
				Lifecycle: MeshTenantLifecycle{State: TenantLifecycleStateActive},
			}},
			wantDone: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			done, err := tt.tenant.DeletionSuccessful()
			assert.Equal(t, tt.wantDone, done)
			assert.NoError(t, err)
		})
	}
}

func TestTenantDeletionState(t *testing.T) {
	assert.Equal(t, tenantNotObserved, (*MeshTenant)(nil).DeletionState())
	assert.Equal(t, tenantNotObserved, (*MeshTenantV4)(nil).DeletionState())

	assert.Equal(t, "DELETED",
		(&MeshTenant{Status: MeshTenantStatus{
			Lifecycle: MeshTenantLifecycle{State: TenantLifecycleStateDeleted},
		}}).DeletionState(),
	)
	assert.Contains(t,
		(&MeshTenant{Status: MeshTenantStatus{Lifecycle: MeshTenantLifecycle{
			State:             TenantLifecycleStateMarkedForDeletion,
			MarkedForDeletion: &MeshTenantLifecycleAction{Timestamp: "2026-07-30T16:14:14Z"},
		}}}).DeletionState(),
		"MARKED_FOR_DELETION since 2026-07-30T16:14:14Z",
	)
	assert.Contains(t,
		(&MeshTenantV4{Status: MeshTenantV4Status{
			Lifecycle: MeshTenantLifecycle{State: TenantLifecycleStateMarkedForDeletion},
		}}).DeletionState(),
		"MARKED_FOR_DELETION, awaiting",
	)
	assert.Contains(t,
		(&MeshTenant{Status: MeshTenantStatus{
			Lifecycle: MeshTenantLifecycle{State: TenantLifecycleStateActive},
		}}).DeletionState(),
		"has not acted on it",
	)
}

// TestMeshTenantV4_UnmarshalJSON_RefShapedSpec covers the meshTenant v4 payload after meshStack
// dropped the deprecated flat identifiers. Both attributes force replacement on
// meshstack_tenant_v4, so reading them back empty would make a refresh plan the recreation of a
// live tenant.
func TestMeshTenantV4_UnmarshalJSON_RefShapedSpec(t *testing.T) {
	const refShaped = `{
	  "metadata": {
	    "uuid": "124b09ec-63b8-452e-a837-44afb382d5bd",
	    "ownedByWorkspace": "smoke-test",
	    "ownedByProject": "smoke-test-20260708163151-dev"
	  },
	  "spec": {
	    "platformRef": { "uuid": "403af12b-fbd5-41f4-aad2-b8c5311bc651", "kind": "meshPlatform" },
	    "landingZoneRef": { "name": "smoketest-ske-dev", "kind": "meshLandingZone" },
	    "platformTenantId": "smoke-test-smoke-test-20260708163151-dev"
	  },
	  "status": {
	    "tenantName": "smoke-test.smoke-test-20260708163151-dev.smoke-test-ske-platform.global"
	  }
	}`

	var tenant MeshTenantV4
	require.NoError(t, json.Unmarshal([]byte(refShaped), &tenant))

	assert.Equal(t, "smoke-test-ske-platform.global", tenant.Spec.PlatformIdentifier,
		"platform identifier is recovered from status.tenantName")
	require.NotNil(t, tenant.Spec.LandingZoneIdentifier)
	assert.Equal(t, "smoketest-ske-dev", *tenant.Spec.LandingZoneIdentifier,
		"landing zone identifier is recovered from spec.landingZoneRef.name")
}

func TestMeshTenantV4_UnmarshalJSON_KeepsFlatIdentifiersWhenPresent(t *testing.T) {
	const flat = `{
	  "metadata": { "ownedByWorkspace": "ws", "ownedByProject": "proj" },
	  "spec": {
	    "platformIdentifier": "flat-platform.global",
	    "landingZoneIdentifier": "flat-lz",
	    "landingZoneRef": { "name": "ref-lz", "kind": "meshLandingZone" }
	  },
	  "status": { "tenantName": "ws.proj.tenant-name-platform.global" }
	}`

	var tenant MeshTenantV4
	require.NoError(t, json.Unmarshal([]byte(flat), &tenant))

	assert.Equal(t, "flat-platform.global", tenant.Spec.PlatformIdentifier)
	require.NotNil(t, tenant.Spec.LandingZoneIdentifier)
	assert.Equal(t, "flat-lz", *tenant.Spec.LandingZoneIdentifier)
}

func TestPlatformIdentifierFromTenantName(t *testing.T) {
	tests := []struct {
		name       string
		tenantName string
		workspace  string
		project    string
		want       string
	}{
		{
			name:       "platform identifier contains dots",
			tenantName: "ws.proj.platform-name.global",
			workspace:  "ws",
			project:    "proj",
			want:       "platform-name.global",
		},
		{
			name:       "workspace and project contain dots",
			tenantName: "my.ws.my.proj.platform.global",
			workspace:  "my.ws",
			project:    "my.proj",
			want:       "platform.global",
		},
		{
			// A name that does not carry the expected prefix yields "" rather than a wrong platform.
			name:       "prefix does not match",
			tenantName: "other.tenant.platform.global",
			workspace:  "ws",
			project:    "proj",
			want:       "",
		},
		{
			name:       "empty tenant name",
			tenantName: "",
			workspace:  "ws",
			project:    "proj",
			want:       "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, platformIdentifierFromTenantName(tt.tenantName, tt.workspace, tt.project))
		})
	}
}
