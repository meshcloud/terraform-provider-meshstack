data "meshstack_workspace_user_binding" "example" {
  metadata = {
    name = "my-workspace-user-binding"
  }
  # or look it up by workspace, user and role:
  # target_ref = { name = "my-workspace" }
  # subject    = { name = "user@meshcloud.io" }
  # role_ref   = { name = "Workspace Member" }
}
