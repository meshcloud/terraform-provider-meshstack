package testconfig

import "testing"

func BuildIntegrationConfig(t *testing.T, suffix string) (config Config, integrationAddr Traversal) {
	t.Helper()
	workspaceConfig, workspaceAddr := BuildWorkspaceConfig(t)
	return Resource{Name: "integration", Suffix: suffix}.Config(t).WithFirstBlock(t,
		ExtractIdentifier(&integrationAddr),
		OwnedByWorkspace(t, workspaceAddr),
	).Join(workspaceConfig), integrationAddr
}
