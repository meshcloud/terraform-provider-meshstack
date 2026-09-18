package provider

import (
	"context"
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/meshcloud/meshstack-cli/client"
	"github.com/meshcloud/meshstack-cli/pkg/auth"
	"github.com/meshcloud/meshstack-cli/pkg/setting"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_workspaceSource(t *testing.T) {
	t.Parallel()

	workspace := func(name string) client.MeshWorkspace {
		return client.MeshWorkspace{Metadata: client.MeshWorkspaceMetadata{Name: name}}
	}
	several := setting.Workspaces{Items: []client.MeshWorkspace{workspace("first-ab12c"), workspace("second-de34f")}}

	tests := []struct {
		name          string
		method        auth.Method
		methodErr     error
		workspaces    setting.Workspaces
		workspacesErr error
		want          string
		wantLookup    bool
	}{
		{
			name:       "a browser login takes the only workspace it reaches",
			method:     auth.OidcLoginMethod,
			workspaces: setting.Workspaces{Items: []client.MeshWorkspace{workspace("only-ab12c")}},
			want:       "only-ab12c",
			wantLookup: true,
		},
		{
			name:       "a browser login takes the profile's default workspace",
			method:     auth.OidcLoginMethod,
			workspaces: setting.Workspaces{Items: several.Items, ProfileDefaultWorkspace: "second-de34f"},
			want:       "second-de34f",
			wantLookup: true,
		},
		{
			name:       "a browser login reaching several workspaces without a default takes none",
			method:     auth.OidcLoginMethod,
			workspaces: several,
			wantLookup: true,
		},
		{
			name:   "an API key lists no workspaces",
			method: auth.ApiKeyMethod,
		},
		{
			name:   "an API token lists no workspaces",
			method: auth.ManualMethod,
		},
		{
			name:      "an unknown credential lists no workspaces",
			methodErr: fmt.Errorf("resolving another setting"),
		},
		{
			name:          "a failed listing leaves the workspace unset",
			method:        auth.OidcLoginMethod,
			workspacesErr: fmt.Errorf("cannot list workspaces; try logging into meshPanel UI first"),
			wantLookup:    true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			lookedUp := false
			source := workspaceSource(
				func(context.Context) (auth.Method, error) {
					return tt.method, tt.methodErr
				},
				func(context.Context) (setting.Workspaces, error) {
					lookedUp = true
					return tt.workspaces, tt.workspacesErr
				},
			)

			got, err := source.Lookup(t.Context(), setting.Workspace.EnvKey())

			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
			assert.Equal(t, tt.wantLookup, lookedUp)
		})
	}
}

// Test_providerSettingSources covers the half of the precedence this repository decides. The
// meshStack CLI ranks a setting.FrontendSource above the environment and a setting.FallbackSource
// below it.
func Test_providerSettingSources(t *testing.T) {
	t.Parallel()

	sources := providerSettingSources(MeshStackProviderModel{Workspace: types.StringValue("named-ab12c")})

	require.Len(t, sources, 2)
	block, ok := sources[0].(setting.FrontendSource)
	require.Truef(t, ok, "the provider block is a %T", sources[0])
	_, ok = sources[1].(setting.FallbackSource)
	require.Truef(t, ok, "the login's workspace is a %T", sources[1])

	fromBlock, err := block.Lookup(t.Context(), setting.Workspace.EnvKey())

	require.NoError(t, err)
	assert.Equal(t, "named-ab12c", fromBlock)
}
