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

// BuildingBlockRunnerWif builds the published workload identity federation example, whose runner
// declares a subject template.
func BuildingBlockRunnerWif(t *testing.T, workspaceAddress Traversal) (config Config, buildingBlockRunnerAddress Traversal) {
	t.Helper()
	return Resource{Name: "building_block_runner", Suffix: "_wif"}.Config(t).WithFirstBlock(
		ExtractAddress(&buildingBlockRunnerAddress),
		OwnedByWorkspace(workspaceAddress),
	), buildingBlockRunnerAddress
}

// BuildingBlockRunnerWifAndWorkspace creates a workspace and a runner with workload identity
// federation owned by it.
func BuildingBlockRunnerWifAndWorkspace(t *testing.T) (config Config, buildingBlockRunnerAddress Traversal, workspaceAddress Traversal) {
	t.Helper()
	workspaceConfig, workspaceAddress := Workspace(t)
	runnerConfig, buildingBlockRunnerAddress := BuildingBlockRunnerWif(t, workspaceAddress)
	return runnerConfig.Join(workspaceConfig), buildingBlockRunnerAddress, workspaceAddress
}
