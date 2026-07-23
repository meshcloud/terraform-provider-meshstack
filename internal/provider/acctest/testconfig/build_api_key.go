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

// OtherProviderConfig loads the shared test-support config declaring the apikey_client_id /
// apikey_client_secret variables and the `meshstack-other` provider alias (backed by a restricted
// API key). Cross-workspace acceptance tests use it to read meshObjects as a second workspace.
func OtherProviderConfig(t *testing.T) Config {
	t.Helper()
	return Resource{Name: "api_key"}.TestSupportConfig(t, "_other_provider")
}
