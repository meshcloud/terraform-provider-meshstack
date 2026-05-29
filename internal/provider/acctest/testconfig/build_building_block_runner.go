package testconfig

import "testing"

// BuildingBlockRunner builds a building block runner config owned by the given workspace.
func BuildingBlockRunner(t *testing.T, workspaceAddress Traversal) (config Config, buildingBlockRunnerAddress Traversal) {
	t.Helper()
	return Resource{Name: "building_block_runner"}.Config(t).WithFirstBlock(
		ExtractAddress(&buildingBlockRunnerAddress),
		OwnedByWorkspace(workspaceAddress),
	), buildingBlockRunnerAddress
}

// BuildingBlockRunnerAndWorkspace creates a workspace and a building block runner owned by it.
func BuildingBlockRunnerAndWorkspace(t *testing.T) (config Config, buildingBlockRunnerAddress Traversal, workspaceAddress Traversal) {
	t.Helper()
	workspaceConfig, workspaceAddress := Workspace(t)
	runnerConfig, buildingBlockRunnerAddress := BuildingBlockRunner(t, workspaceAddress)
	return runnerConfig.Join(workspaceConfig), buildingBlockRunnerAddress, workspaceAddress
}
