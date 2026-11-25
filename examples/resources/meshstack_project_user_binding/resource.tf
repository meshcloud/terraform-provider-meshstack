resource "meshstack_project_user_binding" "example" {
  metadata = {
    name = "example-project-user-binding"
  }

  role_ref = {
    name = "Project Reader"
  }

  target_ref = meshstack_project.example.ref

  # The subject must be a member of the workspace first.
  subject = {
    name = meshstack_workspace_user_binding.example.subject.name
  }

  # Optional. If omitted, the binding never expires.
  expiry_date = "2099-12-31"
}
