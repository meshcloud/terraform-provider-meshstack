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

// BuildingBlockRunnerDataSource reads the runner at the given address. FirstBlockOnly drops the
// example's locals block, which illustrates the backplane wiring for the docs.
func BuildingBlockRunnerDataSource(t *testing.T, buildingBlockRunnerAddress Traversal) (config Config, dataSourceAddress Traversal) {
	t.Helper()
	return DataSource{Name: "building_block_runner"}.Config(t).FirstBlockOnly().WithFirstBlock(
		ExtractAddress(&dataSourceAddress),
		Descend("metadata", "uuid")(SetAddr(buildingBlockRunnerAddress, "metadata", "uuid")),
	), dataSourceAddress
}

// SharedBuildingBlockRunnerDataSource drops metadata to read the shared runner, the documented
// default of an omitted metadata.uuid.
func SharedBuildingBlockRunnerDataSource(t *testing.T) (config Config, dataSourceAddress Traversal) {
	t.Helper()
	return DataSource{Name: "building_block_runner"}.Config(t).FirstBlockOnly().WithFirstBlock(
		ExtractAddress(&dataSourceAddress),
		Descend("metadata")(RemoveKey()),
	), dataSourceAddress
}
