data "meshstack_project_user_binding" "example" {
  metadata = {
    name = "my-project-user-binding"
  }
  # or look it up by project, user and role:
  # target_ref = { owned_by_workspace = "my-customer", name = "my-project" }
  # subject    = { name = "user@meshcloud.io" }
  # role_ref   = { name = "Project Reader" }
}
