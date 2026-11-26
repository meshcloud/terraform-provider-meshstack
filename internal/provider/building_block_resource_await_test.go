package provider

import (
	"context"
	nethttp "net/http"
	"testing"
	"time"

	"github.com/meshcloud/meshstack-cli/client"
	"github.com/stretchr/testify/require"
)

type conflictingDeleteClient struct {
	client.MeshBuildingBlockV2Client
	conflicts int
	deletes   int
}

func (c *conflictingDeleteClient) Delete(context.Context, string) error {
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
