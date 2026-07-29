package testconfig

import (
	"testing"
)

// WorkspaceTags builds a meshstack_workspace_tags config wired to the given workspace address.
// It creates a single tag drawn from a TagDefinition and returns the resource address.
func WorkspaceTags(t *testing.T, workspaceAddr Traversal) (config Config, workspaceTagsAddr Traversal) {
	t.Helper()
	tagConfig, tagDefinitionAddr, _ := TagDefinition(t, "meshWorkspace")
	return Resource{Name: "workspace_tags"}.Config(t).WithFirstBlock(
		ExtractAddress(&workspaceTagsAddr),
		Descend("metadata", "workspace_identifier")(SetAddr(workspaceAddr, "metadata", "name")),
		Descend("spec", "tags")(SetRawExpr(`{(%s) = ["12345"]}`, tagDefinitionAddr.Join("spec", "key"))),
	).Join(tagConfig), workspaceTagsAddr
}
