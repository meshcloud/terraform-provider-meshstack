package provider

import (
	"context"
	"strings"
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

// TestAwaitRunPollsPendingToSuccess: a triggered run surfaces immediately as PENDING (the backend
// eager-sets it when a run will follow), then progresses through IN_PROGRESS to SUCCEEDED. awaitRun polls
// the status to completion — no run-uuid comparison, no grace window — without surfacing a warning.
func TestAwaitRunPollsPendingToSuccess(t *testing.T) {
	t.Parallel()

	stub := &sequencedBBClient{states: []*client.MeshBuildingBlockV2{
		bbWithRun(client.BuildingBlockStatusPending, "run-new"),
		bbWithRun(client.BuildingBlockStatusInProgress, "run-new"),
		bbWithRun(client.BuildingBlockStatusSucceeded, "run-new"),
	}}
	r := &buildingBlockResource{BuildingBlockClient: stub}

	var diags diag.Diagnostics
	final := r.awaitRun(context.Background(), &diags, "bb-uuid", true, 30*time.Second)

	require.False(t, diags.HasError(), "unexpected error diagnostics: %v", diags.Errors())
	require.Empty(t, diags.Warnings(), "a completed run must not surface a waiting-for-input warning")
	require.GreaterOrEqual(t, stub.reads, 3, "must poll through PENDING/IN_PROGRESS to the terminal state")
	require.NotNil(t, final)
	require.Equal(t, client.BuildingBlockStatusSucceeded, final.Status.Status)
}

// TestAwaitRunAwaitsWithoutRunUuid: awaiting keys off the status alone, so a run whose uuids are null
// (low run transparency / insufficient permissions) is still awaited to completion.
func TestAwaitRunAwaitsWithoutRunUuid(t *testing.T) {
	t.Parallel()

	stub := &sequencedBBClient{states: []*client.MeshBuildingBlockV2{
		bbWithStatus(client.BuildingBlockStatusPending),
		bbWithStatus(client.BuildingBlockStatusSucceeded),
	}}
	r := &buildingBlockResource{BuildingBlockClient: stub}

	var diags diag.Diagnostics
	final := r.awaitRun(context.Background(), &diags, "bb-uuid", true, 30*time.Second)

	require.False(t, diags.HasError(), "unexpected error diagnostics: %v", diags.Errors())
	require.Empty(t, diags.Warnings())
	require.GreaterOrEqual(t, stub.reads, 2, "must poll to the terminal state even without a run uuid")
	require.NotNil(t, final)
	require.Equal(t, client.BuildingBlockStatusSucceeded, final.Status.Status)
}

// TestAwaitRunErrorsWhenBlockDisappears: if the block 404s mid-poll (purged, or its definition deleted
// out-of-band), ReadFunc -> Get returns (nil, nil). awaitRun must surface a clear error rather than
// panicking on a nil-block dereference in the poll predicate.
func TestAwaitRunErrorsWhenBlockDisappears(t *testing.T) {
	t.Parallel()

	stub := &sequencedBBClient{states: []*client.MeshBuildingBlockV2{
		bbWithStatus(client.BuildingBlockStatusInProgress),
		nil, // the block 404'd mid-run
	}}
	r := &buildingBlockResource{BuildingBlockClient: stub}

	var diags diag.Diagnostics
	final := r.awaitRun(context.Background(), &diags, "bb-uuid", true, 30*time.Second)

	require.True(t, diags.HasError(), "a block disappearing mid-run must surface an error diagnostic, not panic")
	require.Nil(t, final)
}

// TestAwaitRunSurfacesFailedStepLogAsError: when a run fails and its logs are readable (run transparency
// on / sufficient permissions), awaitRun surfaces the failing step's message as an error diagnostic — the
// actionable detail behind the run failure the apply already reports — rather than a separate warning.
func TestAwaitRunSurfacesFailedStepLogAsError(t *testing.T) {
	t.Parallel()

	stub := &sequencedBBClient{states: []*client.MeshBuildingBlockV2{
		bbWithRun(client.BuildingBlockStatusInProgress, "run-broken"),
		bbWithRun(client.BuildingBlockStatusFailed, "run-broken"),
	}}
	runClient := stubRunLogsClient{logs: client.MeshBuildingBlockRunLogs{Steps: []client.MeshBuildingBlockRunStepLog{
		{DisplayName: "apply", Status: string(client.BuildingBlockStatusFailed), UserMessage: new("intentionally broken BBD version")},
	}}}
	r := &buildingBlockResource{BuildingBlockClient: stub, BuildingBlockRunClient: runClient}

	var diags diag.Diagnostics
	final := r.awaitRun(context.Background(), &diags, "bb-uuid", true, 30*time.Second)

	require.True(t, diags.HasError(), "a failed run must surface error diagnostics")
	require.Empty(t, diags.Warnings(), "readable failed-step logs must be errors, not warnings")
	var foundStepError bool
	for _, e := range diags.Errors() {
		if strings.Contains(e.Detail(), "intentionally broken BBD version") {
			foundStepError = true
		}
	}
	require.True(t, foundStepError, "the failing step's log message must appear in an error diagnostic")
	require.NotNil(t, final)
	require.Equal(t, client.BuildingBlockStatusFailed, final.Status.Status)
}

// TestAwaitRunWarnsWhenFailedRunLogsUnreadable: when a run fails but its logs cannot be read (run
// transparency off / insufficient permissions surface a null run uuid), awaitRun reports the run failure
// as an error and does not attempt to fetch logs, so no step-log error is added.
func TestAwaitRunWarnsWhenFailedRunLogsUnreadable(t *testing.T) {
	t.Parallel()

	stub := &sequencedBBClient{states: []*client.MeshBuildingBlockV2{
		bbWithStatus(client.BuildingBlockStatusInProgress),
		bbWithStatus(client.BuildingBlockStatusFailed), // no run uuid: logs not exposed
	}}
	r := &buildingBlockResource{BuildingBlockClient: stub}

	var diags diag.Diagnostics
	final := r.awaitRun(context.Background(), &diags, "bb-uuid", true, 30*time.Second)

	require.True(t, diags.HasError(), "a failed run must surface an error diagnostic")
	require.NotNil(t, final)
	require.Equal(t, client.BuildingBlockStatusFailed, final.Status.Status)
}

// recordingRunLogsClient records whether GetLogs was called and returns a single failed step whose system
// message identifies which client answered, so a test can assert which client was used to fetch logs.
type recordingRunLogsClient struct {
	client.MeshBuildingBlockRunClient
	systemMessage string
	called        bool
}

func (c *recordingRunLogsClient) GetLogs(_ context.Context, _ string) (client.MeshBuildingBlockRunLogs, error) {
	c.called = true
	return client.MeshBuildingBlockRunLogs{Steps: []client.MeshBuildingBlockRunStepLog{
		{DisplayName: "apply", Status: string(client.BuildingBlockStatusFailed), SystemMessage: new(c.systemMessage)},
	}}, nil
}

func failedRunStates(runUuid string) []*client.MeshBuildingBlockV2 {
	return []*client.MeshBuildingBlockV2{
		bbWithRun(client.BuildingBlockStatusInProgress, runUuid),
		bbWithRun(client.BuildingBlockStatusFailed, runUuid),
	}
}

// TestAwaitRunFetchesFailureLogsWithRunTokenClient: when a run-token client is configured, awaitRun reads
// the failed run's logs with it (not the workspace client) so a composition can surface a child's system
// message even with the child's run transparency disabled.
func TestAwaitRunFetchesFailureLogsWithRunTokenClient(t *testing.T) {
	t.Parallel()

	stub := &sequencedBBClient{states: failedRunStates("run-broken")}
	workspace := &recordingRunLogsClient{systemMessage: "workspace-client logs"}
	runToken := &recordingRunLogsClient{systemMessage: "run-token-client logs"}
	r := &buildingBlockResource{BuildingBlockClient: stub, BuildingBlockRunClient: workspace, RunTokenRunClient: runToken}

	var diags diag.Diagnostics
	r.awaitRun(context.Background(), &diags, "bb-uuid", true, 30*time.Second)

	require.True(t, runToken.called, "the run-token client must fetch failure logs when configured")
	require.False(t, workspace.called, "the workspace client must not be used when the run-token client is configured")
	require.Contains(t, allErrorDetails(diags), "run-token-client logs")
}

// TestAwaitRunFetchesFailureLogsWithWorkspaceClientWithoutRunToken: without a run-token client (no
// MESHSTACK_RUN_TOKEN), awaitRun falls back to the workspace client for the failure logs.
func TestAwaitRunFetchesFailureLogsWithWorkspaceClientWithoutRunToken(t *testing.T) {
	t.Parallel()

	stub := &sequencedBBClient{states: failedRunStates("run-broken")}
	workspace := &recordingRunLogsClient{systemMessage: "workspace-client logs"}
	r := &buildingBlockResource{BuildingBlockClient: stub, BuildingBlockRunClient: workspace}

	var diags diag.Diagnostics
	r.awaitRun(context.Background(), &diags, "bb-uuid", true, 30*time.Second)

	require.True(t, workspace.called, "the workspace client must fetch failure logs when no run-token client is set")
	require.Contains(t, allErrorDetails(diags), "workspace-client logs")
}

func allErrorDetails(diags diag.Diagnostics) string {
	var b strings.Builder
	for _, e := range diags.Errors() {
		b.WriteString(e.Detail())
		b.WriteByte('\n')
	}
	return b.String()
}

// TestAwaitRunWarnsOnParkedWaiting: a block parked in WAITING_FOR_*_INPUT cannot proceed from this apply
// (a runnable block would already be PENDING). awaitRun returns it as terminal-but-non-fatal with a
// waiting-for-input warning, rather than polling to the timeout.
func TestAwaitRunWarnsOnParkedWaiting(t *testing.T) {
	t.Parallel()

	stub := &sequencedBBClient{states: []*client.MeshBuildingBlockV2{
		bbWithStatus(client.BuildingBlockStatusWaitingForOperatorInput),
	}}
	r := &buildingBlockResource{BuildingBlockClient: stub}

	var diags diag.Diagnostics
	final := r.awaitRun(context.Background(), &diags, "bb-uuid", true, 30*time.Second)

	require.False(t, diags.HasError(), "unexpected error diagnostics: %v", diags.Errors())
	require.NotEmpty(t, diags.Warnings(), "a parked WAITING block must surface a waiting-for-input warning")
	require.NotNil(t, final)
	require.Equal(t, client.BuildingBlockStatusWaitingForOperatorInput, final.Status.Status)
}
