package provider

import (
	"context"
	"testing"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/stretchr/testify/require"

	"github.com/meshcloud/terraform-provider-meshstack/client"
	"github.com/meshcloud/terraform-provider-meshstack/client/types/enum"
)

// stubRunLogsClient is a stub MeshBuildingBlockRunClient returning canned logs/error for GetLogs.
// awaitRun's failure path calls only GetLogs, so the embedded interface stays nil.
type stubRunLogsClient struct {
	client.MeshBuildingBlockRunClient
	logs client.MeshBuildingBlockRunLogs
	err  error
}

func (c stubRunLogsClient) GetLogs(_ context.Context, _ string) (client.MeshBuildingBlockRunLogs, error) {
	return c.logs, c.err
}

// sequencedBBClient is a stub MeshBuildingBlockV2Client whose Read returns a queued sequence of
// building block states (repeating the last once exhausted). awaitRun only calls Read/ReadFunc, so
// the embedded interface stays nil and would panic if any other method were unexpectedly invoked.
type sequencedBBClient struct {
	client.MeshBuildingBlockV2Client
	states []*client.MeshBuildingBlockV2
	reads  int
}

func (c *sequencedBBClient) Read(_ context.Context, _ string) (*client.MeshBuildingBlockV2, error) {
	state := c.states[min(c.reads, len(c.states)-1)]
	c.reads++
	return state, nil
}

func (c *sequencedBBClient) ReadFunc(uuid string) func(context.Context) (*client.MeshBuildingBlockV2, error) {
	return func(ctx context.Context) (*client.MeshBuildingBlockV2, error) { return c.Read(ctx, uuid) }
}

func bbWithRun(status enum.Entry[client.BuildingBlockStatus], runUuid string) *client.MeshBuildingBlockV2 {
	return &client.MeshBuildingBlockV2{
		Status: &client.MeshBuildingBlockV2Status{
			Status:        status,
			LatestRunUuid: &runUuid,
		},
	}
}

// bbWithStatus builds a building block carrying only a status and no run uuid — the backend leaves the run
// uuids null when run transparency / permissions do not expose them, and awaitRun must still work.
func bbWithStatus(status enum.Entry[client.BuildingBlockStatus]) *client.MeshBuildingBlockV2 {
	return &client.MeshBuildingBlockV2{
		Status: &client.MeshBuildingBlockV2Status{Status: status},
	}
}

// TestAwaitRun pins how awaitRun reports each terminal building block status. A run triggered by the
// preceding create/update surfaces immediately as PENDING, so awaiting keys off the status alone.
//
// Every terminal status that is not SUCCEEDED is a warning: failing the apply is the configuration's
// decision, taken with a postcondition. Only a run whose outcome could not be established at all — a
// timeout, an unknown status, a block that vanished — is an error.
func TestAwaitRun(t *testing.T) {
	t.Parallel()

	failedStep := client.MeshBuildingBlockRunLogs{Steps: []client.MeshBuildingBlockRunStepLog{
		{DisplayName: "apply", Status: string(client.BuildingBlockStatusFailed), UserMessage: new("intentionally broken BBD version")},
	}}

	tests := map[string]struct {
		states       []*client.MeshBuildingBlockV2
		logs         *stubRunLogsClient
		wantErrors   []string // one substring per expected error diagnostic, in order
		wantWarnings []string // one substring per expected warning diagnostic, in order
		wantStatus   enum.Entry[client.BuildingBlockStatus]
		wantNoBlock  bool
		wantReads    int
	}{
		"a run is polled through PENDING and IN_PROGRESS to SUCCEEDED": {
			states: []*client.MeshBuildingBlockV2{
				bbWithRun(client.BuildingBlockStatusPending, "run-new"),
				bbWithRun(client.BuildingBlockStatusInProgress, "run-new"),
				bbWithRun(client.BuildingBlockStatusSucceeded, "run-new"),
			},
			wantStatus: client.BuildingBlockStatusSucceeded,
			wantReads:  3,
		},
		"a block returned without a status yet is polled on": {
			states: []*client.MeshBuildingBlockV2{
				{}, // the backend has not attached a status yet
				bbWithStatus(client.BuildingBlockStatusSucceeded),
			},
			wantStatus: client.BuildingBlockStatusSucceeded,
			wantReads:  2,
		},
		"a run without a run uuid is awaited all the same": {
			states: []*client.MeshBuildingBlockV2{
				bbWithStatus(client.BuildingBlockStatusPending),
				bbWithStatus(client.BuildingBlockStatusSucceeded),
			},
			wantStatus: client.BuildingBlockStatusSucceeded,
			wantReads:  2,
		},
		"a failed run and its failing step are warned about": {
			states: []*client.MeshBuildingBlockV2{
				bbWithRun(client.BuildingBlockStatusInProgress, "run-broken"),
				bbWithRun(client.BuildingBlockStatusFailed, "run-broken"),
			},
			logs:         &stubRunLogsClient{logs: failedStep},
			wantWarnings: []string{"ended in status FAILED", "intentionally broken BBD version"},
			wantStatus:   client.BuildingBlockStatusFailed,
		},
		"a failed run whose logs are not exposed reports the failure alone": {
			states: []*client.MeshBuildingBlockV2{
				bbWithStatus(client.BuildingBlockStatusInProgress),
				bbWithStatus(client.BuildingBlockStatusFailed), // no run uuid: logs not exposed
			},
			wantWarnings: []string{"ended in status FAILED"},
			wantStatus:   client.BuildingBlockStatusFailed,
		},
		"a failed run whose logs cannot be read warns about the logs": {
			states: []*client.MeshBuildingBlockV2{
				bbWithRun(client.BuildingBlockStatusFailed, "run-broken"),
			},
			logs:         &stubRunLogsClient{err: context.DeadlineExceeded},
			wantWarnings: []string{"ended in status FAILED", "Could not fetch run logs"},
			wantStatus:   client.BuildingBlockStatusFailed,
		},
		// An aborted run is reported exactly like a failed one: the building block is off its
		// configuration either way, and the next plan runs it again either way.
		"an aborted run is warned about like a failed one": {
			states: []*client.MeshBuildingBlockV2{
				bbWithRun(client.BuildingBlockStatusInProgress, "run-stopped"),
				bbWithRun(client.BuildingBlockStatusAborted, "run-stopped"),
			},
			logs:         &stubRunLogsClient{}, // an aborted run has no failed step to report
			wantWarnings: []string{"ended in status ABORTED"},
			wantStatus:   client.BuildingBlockStatusAborted,
		},
		"a block parked for input warns instead of polling to the timeout": {
			states:       []*client.MeshBuildingBlockV2{bbWithStatus(client.BuildingBlockStatusWaitingForOperatorInput)},
			wantWarnings: []string{"waiting for input or approval"},
			wantStatus:   client.BuildingBlockStatusWaitingForOperatorInput,
		},
		"a status this provider does not know errors instead of polling to the timeout": {
			states:     []*client.MeshBuildingBlockV2{bbWithStatus("SOMETHING_NEW")},
			wantErrors: []string{"unknown building block status"},
			wantStatus: "SOMETHING_NEW",
		},
		// A block that vanished left no status for a postcondition to decide on, so it stays an error.
		"a block that disappears mid-run errors": {
			states: []*client.MeshBuildingBlockV2{
				bbWithStatus(client.BuildingBlockStatusInProgress),
				nil, // the block 404'd mid-run
			},
			wantErrors:  []string{"disappeared while waiting"},
			wantNoBlock: true,
		},
	}

	requireDiagnostics := func(t *testing.T, got diag.Diagnostics, want []string, kind string) {
		t.Helper()
		require.Len(t, got, len(want), "unexpected %s diagnostics: %v", kind, got)
		for i, substring := range want {
			require.Contains(t, got[i].Summary()+"\n"+got[i].Detail(), substring)
		}
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			stub := &sequencedBBClient{states: tt.states}
			r := &buildingBlockResource{BuildingBlockClient: stub}
			if tt.logs != nil {
				r.BuildingBlockRunClient = *tt.logs
			}
			var diags diag.Diagnostics
			final := r.awaitRun(context.Background(), &diags, "bb-uuid", true, 30*time.Second)

			requireDiagnostics(t, diags.Errors(), tt.wantErrors, "error")
			requireDiagnostics(t, diags.Warnings(), tt.wantWarnings, "warning")
			if tt.wantNoBlock {
				require.Nil(t, final)
				return
			}
			require.NotNil(t, final)
			require.Equal(t, tt.wantStatus, final.Status.Status)
			if tt.wantReads > 0 {
				require.GreaterOrEqual(t, stub.reads, tt.wantReads, "must poll to the terminal state")
			}
		})
	}
}
