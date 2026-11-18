resource "meshstack_workspace" "test" {
  metadata = {
    name = "test-ws-[[.suffix]]"
  }
  spec = {
    display_name = "Test Workspace"
  }
}

data "meshstack_workspace" "example" {
  metadata = {
    name = meshstack_workspace.test.metadata.name
  }
}

data "meshstack_users" "all" {}

output "seeded_user_listed" {
  value = contains(data.meshstack_users.all.users[*].metadata.name, "[[.seeded_username]]")
}

[[ example "data-sources/meshstack_users/data-source_workspace.tf" ]]
