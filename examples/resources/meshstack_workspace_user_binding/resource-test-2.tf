resource "meshstack_workspace_user_binding" "example" {
  metadata = {
    name = "test-wub-${var.suffix}"
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

  # expiry_date omitted: the Optional+Computed attribute stays unknown in the plan, which used to
  # fail the create with a "Received unknown value" conversion error (#267, #293).
}
