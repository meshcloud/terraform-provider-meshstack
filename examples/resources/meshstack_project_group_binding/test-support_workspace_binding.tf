resource "meshstack_workspace_group_binding" "project_subject" {
  metadata = {
    name = "my-workspace-group-binding"
  }

  role_ref = {
    name = "Workspace Member"
  }

  target_ref = {
    name = "my-customer"
  }

  subject = {
    name = "my-user-group"
  }
}
