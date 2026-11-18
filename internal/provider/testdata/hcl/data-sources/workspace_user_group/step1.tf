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

data "meshstack_user" "example" {
  metadata = {
    name = "[[.seeded_username]]"
  }
}

resource "meshstack_workspace_user_group" "test" {
  metadata = {
    name               = "my-user-group"
    owned_by_workspace = data.meshstack_workspace.example.metadata.name
  }
  spec = {
    display_name = "My User Group"
    members      = [data.meshstack_user.example.metadata.name]
  }
}
