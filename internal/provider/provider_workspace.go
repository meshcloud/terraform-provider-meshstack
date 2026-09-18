package provider

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/meshcloud/meshstack-cli/pkg/auth"
	"github.com/meshcloud/meshstack-cli/pkg/setting"
)

func newWorkspaceSource() setting.FallbackSource {
	return workspaceSource(auth.MethodFromContext, setting.WorkspacesFromContext)
}

// workspaceSource picks the workspace a browser login acts in, where nothing above it named one.
// It is the meshStack CLI's workspace selection minus the prompt, and a fallback source for the
// same reason: a profile's stored default ranks below it, because only a login changes that default.
//
// It looks nothing up for an API key or an API token. Both resolve their workspace without reaching
// the backend, and a credential holding no permission to list workspaces would fail there.
//
// It never fails: a configuration that resolves today must not start failing over a lookup.
//
// Both lookups are parameters because the values they read are in the context only while
// MESHSTACK_WORKSPACE resolves, which a test cannot arrange.
func workspaceSource(
	methodFromContext func(context.Context) (auth.Method, error),
	workspacesFromContext func(context.Context) (setting.Workspaces, error),
) setting.FallbackSource {
	return setting.FallbackLookupSource(setting.Workspace.EnvKey(), "the workspace of the profile's login",
		func(ctx context.Context) (string, error) {
			method, err := methodFromContext(ctx)
			if err != nil {
				slog.DebugContext(ctx, fmt.Sprintf("Looking up no workspace, because the credential is unknown: %v", err))
				return "", nil
			}
			if method != auth.OidcLoginMethod {
				return "", nil
			}
			workspaces, err := workspacesFromContext(ctx)
			if err != nil {
				slog.DebugContext(ctx, fmt.Sprintf("Looking up no workspace, because the login's workspaces are unknown: %v", err))
				return "", nil
			}
			selected := workspaces.Single(ctx)
			if selected == nil {
				selected = workspaces.ProfileDefault(ctx)
			}
			if selected == nil {
				return "", nil
			}
			return string(selected.Name()), nil
		})
}
