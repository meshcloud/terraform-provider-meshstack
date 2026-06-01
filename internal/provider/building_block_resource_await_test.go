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

// TestAwaitRunPollsThroughWaiting guards the demo-discovered finding: when an update is expected to
// start a run (expectRun=true) and the block is still parked in WAITING_FOR_OPERATOR_INPUT because the
// asynchronously-created run has not surfaced yet, awaitRun must keep polling until the new run reaches
// a terminal state instead of returning early on the transient WAITING with the waiting-for-input
// warning. The block first reports the pre-update (baseline) run still WAITING, then the new run SUCCEEDED.
func TestAwaitRunPollsThroughWaiting(t *testing.T) {
	t.Parallel()

	stub := &sequencedBBClient{states: []*client.MeshBuildingBlockV2{
		bbWithRun(client.BuildingBlockStatusWaitingForOperatorInput, "run-baseline"),
		bbWithRun(client.BuildingBlockStatusSucceeded, "run-new"),
	}}
	r := &buildingBlockResource{BuildingBlockClient: stub}

	baseline := "run-baseline"
	var diags diag.Diagnostics
	final := r.awaitRun(context.Background(), &diags, "bb-uuid", &baseline, true, true, false, 30*time.Second)

	require.False(t, diags.HasError(), "unexpected error diagnostics: %v", diags.Errors())
	require.Empty(t, diags.Warnings(), "must not surface a waiting-for-input warning once the run completes")
	require.GreaterOrEqual(t, stub.reads, 2, "must poll past the initial WAITING state, not return early")
	require.NotNil(t, final)
	require.Equal(t, client.BuildingBlockStatusSucceeded, final.Status.Status)
}

// TestAwaitRunResumesInPlace guards the resume-in-place case: a block parked in
// WAITING_FOR_OPERATOR_INPUT whose missing input is now supplied resumes the SAME run (its uuid equals
// the pre-update baseline) and reaches SUCCEEDED. With resumeInPlace set, awaitRun must poll through the
// transient WAITING and accept that terminal run despite the matching uuid — the rerun guard ("wait for a
// different run uuid") would otherwise never be satisfied and the poll would run to timeout.
func TestAwaitRunResumesInPlace(t *testing.T) {
	t.Parallel()

	stub := &sequencedBBClient{states: []*client.MeshBuildingBlockV2{
		bbWithRun(client.BuildingBlockStatusWaitingForOperatorInput, "run-parked"),
		bbWithRun(client.BuildingBlockStatusSucceeded, "run-parked"),
	}}
	r := &buildingBlockResource{BuildingBlockClient: stub}

	baseline := "run-parked"
	var diags diag.Diagnostics
	final := r.awaitRun(context.Background(), &diags, "bb-uuid", &baseline, true, true, true, 30*time.Second)

	require.False(t, diags.HasError(), "unexpected error diagnostics: %v", diags.Errors())
	require.Empty(t, diags.Warnings(), "must not warn once the resumed run completes")
	require.GreaterOrEqual(t, stub.reads, 2, "must poll past the initial WAITING state, not return early")
	require.NotNil(t, final)
	require.Equal(t, client.BuildingBlockStatusSucceeded, final.Status.Status)
}
