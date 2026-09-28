resource "meshstack_workspace_user_binding" "example" {
  metadata = {
    name = "this-is-an-example"
  }

  role_ref = {
    name = "Workspace Member"
  }

  target_ref = {
    name = meshstack_workspace.example.metadata.name
  }

  subject = {
    name = "user@meshcloud.io"
  }

  expiry_date = "2026-12-31"
}
