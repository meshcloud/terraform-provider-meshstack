resource "meshstack_workspace_user_binding" "example" {
  metadata = {
    name = "this-is-an-example"
  }

  role_ref = {
    name = "Workspace Member"
  }

  target_ref = {
    name = "my-workspace"
  }

  subject = {
    name = "user@meshcloud.io"
  }

  # Optional. If omitted, the binding never expires. If recertification is enabled
  # for the role, meshStack assigns the maximum allowed expiry date instead.
  expiry_date = "2026-12-31"
}
