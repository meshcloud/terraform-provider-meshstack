package testconfig

import (
	"testing"
)

// WorkspaceTag builds a meshstack_workspace_tag config wired to the given workspace address.
// It creates a single tag drawn from a TagDefinition, returning the resource address and tag key.
func WorkspaceTag(t *testing.T, workspaceAddr Traversal) (config Config, workspaceTagAddr Traversal, tagKey string) {
	t.Helper()
	tagConfig, tagDefinitionAddr, tagKey := TagDefinition(t, "meshWorkspace")
	return Resource{Name: "workspace_tag"}.Config(t).WithFirstBlock(
		ExtractAddress(&workspaceTagAddr),
		Descend("metadata", "workspace_identifier")(SetAddr(workspaceAddr, "metadata", "name")),
		Descend("metadata", "key")(SetAddr(tagDefinitionAddr, "spec", "key")),
		Descend("spec", "values")(SetRawExpr(`["12345"]`)),
	).Join(tagConfig), workspaceTagAddr, tagKey
}

// WorkspaceTagAndWorkspace builds a meshstack_workspace_tag config with a new workspace.
func WorkspaceTagAndWorkspace(t *testing.T) (config Config, workspaceTagAddr Traversal, workspaceAddr Traversal) {
	t.Helper()
	workspaceConfig, workspaceAddr := WorkspaceWithoutTags(t)
	config, workspaceTagAddr, _ = WorkspaceTag(t, workspaceAddr)
	return config.Join(workspaceConfig), workspaceTagAddr, workspaceAddr
}
