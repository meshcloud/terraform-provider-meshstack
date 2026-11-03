data "meshstack_projects" "example_all_projects_in_workspace" {
  workspace_identifier = data.meshstack_workspace.example.metadata.name
}
