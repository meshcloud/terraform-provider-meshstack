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
