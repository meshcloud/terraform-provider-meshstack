resource "meshstack_project_group_binding" "example" {
  metadata = {
    name = "example-project-group-binding"
  }

  role_ref = {
    name = "Project Reader"
  }

  target_ref = {
    owned_by_workspace = meshstack_project.example.metadata.owned_by_workspace
    name               = meshstack_project.example.metadata.name
  }

  # The subject must be a member of the workspace first.
  subject = {
    name = meshstack_workspace_group_binding.example.subject.name
  }

  # Optional. If omitted, the binding never expires.
  expiry_date = "2099-12-31"
}
