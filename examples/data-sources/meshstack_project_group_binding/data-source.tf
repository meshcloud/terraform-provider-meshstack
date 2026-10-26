data "meshstack_project_group_binding" "example" {
  metadata = {
    name = "my-project-group-binding"
  }
  # or look it up by project, group and role:
  # target_ref = { owned_by_workspace = "my-customer", name = "my-project" }
  # subject    = { name = "my-user-group" }
  # role_ref   = { name = "Project Reader" }
}
