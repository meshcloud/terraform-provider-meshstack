data "meshstack_workspace_group_binding" "example" {
  metadata = {
    name = "my-workspace-group-binding"
  }
  # or look it up by workspace, group and role:
  # target_ref = { name = "my-workspace" }
  # subject    = { name = "my-user-group" }
  # role_ref   = { name = "Workspace Member" }
}
