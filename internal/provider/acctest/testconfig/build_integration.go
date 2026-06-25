package testconfig

import "testing"

// Integration builds an integration config with the given suffix for the example resource,
// owned by a freshly created (randomized) test workspace.
func Integration(t *testing.T, suffix string) (config Config, integrationAddr Traversal) {
	t.Helper()
	workspaceConfig, workspaceAddr := Workspace(t)
	return Resource{Name: "integration", Suffix: suffix}.Config(t).WithFirstBlock(
		ExtractAddress(&integrationAddr),
		OwnedByWorkspace(workspaceAddr),
	).Join(workspaceConfig), integrationAddr
}

// IntegrationForWorkspace builds an integration config with the given suffix, owned by an
// already-existing workspace referenced by its literal identifier — no workspace resource is
// created. Use this for integration types meshStack only permits on a specific pre-seeded
// workspace, e.g. Entra ID integrations which must be owned by the admin (partner) workspace.
func IntegrationForWorkspace(t *testing.T, suffix, workspaceName string) (config Config, integrationAddr Traversal) {
	t.Helper()
	return Resource{Name: "integration", Suffix: suffix}.Config(t).WithFirstBlock(
		ExtractAddress(&integrationAddr),
		Descend("metadata", "owned_by_workspace")(SetString(workspaceName)),
	), integrationAddr
}
