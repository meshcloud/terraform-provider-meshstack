data "meshstack_users" "workspace_managers" {
  workspace_identifier = data.meshstack_workspace.example.metadata.name
  workspace_role       = "Workspace Manager"
}
