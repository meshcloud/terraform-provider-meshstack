resource "meshstack_workspace_user_binding" "project_subject" {
  metadata = {
    name = "my-workspace-user-binding"
  }

  role_ref = {
    name = "Workspace Member"
  }

  target_ref = {
    name = "my-customer"
  }

  subject = {
    name = "user@meshcloud.io"
  }
}
