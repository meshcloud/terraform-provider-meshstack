data "meshstack_projects" "example_all_projects_in_workspace" {
  owned_by_workspace = data.meshstack_workspace.example.metadata.name
}
