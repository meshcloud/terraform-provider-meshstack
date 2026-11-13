package provider

import (
	"context"
	"errors"
	"testing"

	"github.com/meshcloud/meshstack-cli/client"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type listRunnersFunc func() ([]client.MeshBuildingBlockRunner, error)

type fakeRunnerLister struct {
	client.MeshBuildingBlockRunnerClient
	list listRunnersFunc
}

func (f fakeRunnerLister) List(context.Context, client.MeshBuildingBlockRunnerFilter) ([]client.MeshBuildingBlockRunner, error) {
	return f.list()
}

func TestDefaultBuildingBlockRunnerUuid(t *testing.T) {
	t.Parallel()

	runner := func(uuid string, isDefault *bool) client.MeshBuildingBlockRunner {
		return client.MeshBuildingBlockRunner{
			Metadata: client.MeshBuildingBlockRunnerMetadata{Uuid: new(uuid)},
			Status:   &client.MeshBuildingBlockRunnerStatus{IsDefault: isDefault},
		}
	}
	returns := func(runners ...client.MeshBuildingBlockRunner) listRunnersFunc {
		return func() ([]client.MeshBuildingBlockRunner, error) { return runners, nil }
	}

	tests := []struct {
		name    string
		list    listRunnersFunc
		want    string
		wantErr string
	}{
		{name: "one default", list: returns(runner("default", new(true))), want: "default"},
		{
			name: "meshStack without status.isDefault ignores the filter",
			list: returns(runner("self-hosted", nil), runner(SharedBuildingBlockRunnerUuid, nil)),
			want: SharedBuildingBlockRunnerUuid,
		},
		{name: "runner without status", list: returns(client.MeshBuildingBlockRunner{}), want: SharedBuildingBlockRunnerUuid},
		{name: "no runner", list: returns(), wantErr: "reports no default building block runner"},
		{name: "two defaults", list: returns(runner("a", new(true)), runner("b", new(true))), wantErr: "reports 2 default building block runners"},
		{
			name:    "list fails",
			list:    func() ([]client.MeshBuildingBlockRunner, error) { return nil, errors.New("forbidden") },
			wantErr: "forbidden",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			got, err := defaultBuildingBlockRunnerUuid(t.Context(), fakeRunnerLister{list: test.list})

			if test.wantErr != "" {
				require.ErrorContains(t, err, test.wantErr)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, test.want, got)
		})
	}
}
