resource "meshstack_workspace_group_binding" "example" {
  metadata = {
    name = "example-workspace-group-binding"
  }

  role_ref = {
    name = "Workspace Member"
  }

  target_ref = {
    name = data.meshstack_workspace.example.metadata.name
  }

  subject = {
    name = "global-group"
  }

  # Optional. If omitted, the binding never expires. If recertification is enabled
  # for the role, meshStack assigns the maximum allowed expiry date instead.
  expiry_date = "2099-12-31"
}
