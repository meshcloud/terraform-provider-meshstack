resource "meshstack_workspace_group_binding" "example" {
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
    name = "my-user-group"
  }

  expiry_date = "2026-12-31"
}
