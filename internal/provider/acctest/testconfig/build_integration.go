package testconfig

import "testing"

// Integration builds an integration config with the given suffix for the example resource.
func Integration(t *testing.T, suffix string) (config Config, integrationAddr Traversal) {
	t.Helper()
	workspaceConfig, workspaceAddr := Workspace(t)
	return Resource{Name: "integration", Suffix: suffix}.Config(t).WithFirstBlock(
		ExtractAddress(&integrationAddr),
		OwnedByWorkspace(workspaceAddr),
	).Join(workspaceConfig), integrationAddr
}
