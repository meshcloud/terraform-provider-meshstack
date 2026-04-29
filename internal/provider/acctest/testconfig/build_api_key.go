package testconfig

import (
	"testing"
)

// ApiKey builds an api_key config owned by the given workspace.
func ApiKey(t *testing.T, workspaceAddr Traversal) (config Config, apiKeyAddr Traversal) {
	t.Helper()
	return Resource{Name: "api_key"}.Config(t).WithFirstBlock(
		ExtractAddress(&apiKeyAddr),
		OwnedByWorkspace(workspaceAddr),
	), apiKeyAddr
}
