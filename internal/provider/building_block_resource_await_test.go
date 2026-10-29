package provider

import (
	"cmp"
	"context"
	nethttp "net/http"
	"testing"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/meshcloud/meshstack-cli/client"
	"github.com/meshcloud/meshstack-cli/client/types/enum"
	"github.com/stretchr/testify/require"
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

func bbWithRun(action enum.Entry[client.BuildingBlockAction], runUuid string) *client.MeshBuildingBlockV2 {
	return &client.MeshBuildingBlockV2{
		Metadata: client.MeshBuildingBlockV2Metadata{Uuid: new("bb-uuid")},
		Status: &client.MeshBuildingBlockV2Status{
			Action:        action,
			LatestRunUuid: &runUuid,
		},
	}
}

func bbWithAction(action enum.Entry[client.BuildingBlockAction]) *client.MeshBuildingBlockV2 {
	return &client.MeshBuildingBlockV2{
		Metadata: client.MeshBuildingBlockV2Metadata{Uuid: new("bb-uuid")},
		Status:   &client.MeshBuildingBlockV2Status{Action: action},
	}
}

func TestAwaitRun(t *testing.T) {
	t.Parallel()

	failedStep := client.MeshBuildingBlockRunLogs{Steps: []client.MeshBuildingBlockRunStepLog{
		{DisplayName: "apply", Status: string(client.BuildingBlockStatusFailed), UserMessage: new("intentionally broken BBD version")},
	}}

	tests := map[string]struct {
		before       *client.MeshBuildingBlockV2Status
		states       []*client.MeshBuildingBlockV2
		timeout      time.Duration
		logs         *stubRunLogsClient
		wantErrors   []string // one substring per expected error diagnostic, in order
		wantWarnings []string // one substring per expected warning diagnostic, in order
		wantAction   enum.Entry[client.BuildingBlockAction]
		wantNoBlock  bool
		wantReads    int
	}{
		"a run is polled through PENDING and IN_PROGRESS to NO_ACTION": {
			states: []*client.MeshBuildingBlockV2{
				bbWithRun(client.BuildingBlockActionPending, "run-new"),
				bbWithRun(client.BuildingBlockActionInProgress, "run-new"),
				bbWithRun(client.BuildingBlockActionNoAction, "run-new"),
			},
			wantAction: client.BuildingBlockActionNoAction,
			wantReads:  3,
		},
		"a block returned without a status yet is polled on": {
			states: []*client.MeshBuildingBlockV2{
				{},
				bbWithAction(client.BuildingBlockActionNoAction),
			},
			wantAction: client.BuildingBlockActionNoAction,
			wantReads:  2,
		},
		"an update whose run meshStack does not start errors": {
			before:     bbWithRun(client.BuildingBlockActionNoAction, "run-before").Status,
			states:     []*client.MeshBuildingBlockV2{bbWithRun(client.BuildingBlockActionNoAction, "run-before")},
			timeout:    time.Second,
			wantErrors: []string{"did not start a run"},
			wantAction: client.BuildingBlockActionNoAction,
		},
		"a failed run and its failing step are warned about": {
			states: []*client.MeshBuildingBlockV2{
				bbWithRun(client.BuildingBlockActionInProgress, "run-broken"),
				bbWithRun(client.BuildingBlockActionReviewErrors, "run-broken"),
			},
			logs:         &stubRunLogsClient{logs: failedStep},
			wantWarnings: []string{"ended with the action REVIEW_ERRORS", "intentionally broken BBD version"},
			wantAction:   client.BuildingBlockActionReviewErrors,
		},
		"a failed run whose logs are not exposed reports the failure alone": {
			states: []*client.MeshBuildingBlockV2{
				bbWithAction(client.BuildingBlockActionInProgress),
				bbWithAction(client.BuildingBlockActionReviewErrors),
			},
			wantWarnings: []string{"ended with the action REVIEW_ERRORS"},
			wantAction:   client.BuildingBlockActionReviewErrors,
		},
		"a failed run whose logs cannot be read warns about the logs": {
			states: []*client.MeshBuildingBlockV2{
				bbWithRun(client.BuildingBlockActionReviewErrors, "run-broken"),
			},
			logs:         &stubRunLogsClient{err: context.DeadlineExceeded},
			wantWarnings: []string{"ended with the action REVIEW_ERRORS", "Could not fetch run logs"},
			wantAction:   client.BuildingBlockActionReviewErrors,
		},
		"an aborted run is warned about like a failed one": {
			states: []*client.MeshBuildingBlockV2{
				bbWithRun(client.BuildingBlockActionInProgress, "run-stopped"),
				bbWithRun(client.BuildingBlockActionReviewAbortedRun, "run-stopped"),
			},
			logs:         &stubRunLogsClient{},
			wantWarnings: []string{"ended with the action REVIEW_ABORTED_RUN"},
			wantAction:   client.BuildingBlockActionReviewAbortedRun,
		},
		"a run that waits for input ends the wait with a warning": {
			states:       []*client.MeshBuildingBlockV2{bbWithAction(client.BuildingBlockActionProvideInput)},
			wantWarnings: []string{"waiting for input or approval"},
			wantAction:   client.BuildingBlockActionProvideInput,
		},
		"an action this provider does not know ends the wait with a warning": {
			states:       []*client.MeshBuildingBlockV2{bbWithAction("SOMETHING_NEW")},
			wantWarnings: []string{"does not know"},
			wantAction:   "SOMETHING_NEW",
		},
		"a run that does not finish in time is warned about": {
			states:       []*client.MeshBuildingBlockV2{bbWithAction(client.BuildingBlockActionInProgress)},
			timeout:      time.Second,
			wantWarnings: []string{"has not finished"},
			wantAction:   client.BuildingBlockActionInProgress,
		},
		"a block that disappears during the run errors": {
			states: []*client.MeshBuildingBlockV2{
				bbWithAction(client.BuildingBlockActionInProgress),
				nil,
			},
			wantErrors:  []string{"disappeared"},
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
			timeout := cmp.Or(tt.timeout, 30*time.Second)
			var diags diag.Diagnostics
			final := r.awaitRun(context.Background(), &diags, "bb-uuid", tt.before, timeout)

			requireDiagnostics(t, diags.Errors(), tt.wantErrors, "error")
			requireDiagnostics(t, diags.Warnings(), tt.wantWarnings, "warning")
			if tt.wantNoBlock {
				require.Nil(t, final)
				return
			}
			require.NotNil(t, final)
			require.Equal(t, tt.wantAction, final.Status.Action)
			require.GreaterOrEqual(t, stub.reads, tt.wantReads, "must poll until the run ended")
		})
	}
}

type conflictingDeleteClient struct {
	client.MeshBuildingBlockV2Client
	conflicts int
	deletes   int
}

func (c *conflictingDeleteClient) Delete(_ context.Context, _ string, _ bool) error {
	c.deletes++
	if c.deletes <= c.conflicts {
		return client.HttpError{StatusCode: nethttp.StatusConflict, ResponseBody: []byte(`{"errorCode":"BuildingBlockConflict"}`)}
	}
	return nil
}

func TestRequestDeletion(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		conflicts   int
		timeout     time.Duration
		wantErr     string
		wantDeletes int
	}{
		"a conflict is waited out until meshStack accepts the delete": {
			conflicts:   1,
			timeout:     30 * time.Second,
			wantDeletes: 2,
		},
		"a conflict that lasts longer than the timeout fails the destroy": {
			conflicts:   100,
			timeout:     time.Second,
			wantErr:     "BuildingBlockConflict",
			wantDeletes: 1,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			stub := &conflictingDeleteClient{conflicts: tt.conflicts}
			r := &buildingBlockResource{BuildingBlockClient: stub}

			err := r.requestDeletion(context.Background(), "bb-uuid", false, tt.timeout)

			if tt.wantErr == "" {
				require.NoError(t, err)
			} else {
				require.ErrorContains(t, err, tt.wantErr)
			}
			require.GreaterOrEqual(t, stub.deletes, tt.wantDeletes)
		})
	}
}
