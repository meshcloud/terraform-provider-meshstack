resource "meshstack_workspace_user_binding" "example" {
  metadata = {
    name = "example-workspace-user-binding"
  }

  role_ref = {
    name = "Workspace Member"
  }

  target_ref = {
    name = data.meshstack_workspace.example.metadata.name
  }

  subject = {
    name = "customer@meshcloud.io"
  }

  # Optional. If omitted, the binding never expires. If recertification is enabled
  # for the role, meshStack assigns the maximum allowed expiry date instead.
  expiry_date = "2099-12-31"
}
