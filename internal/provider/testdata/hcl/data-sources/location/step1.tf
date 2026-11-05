resource "meshstack_workspace" "test" {
  metadata = {
    name = "test-ws-[[.suffix]]"
  }
  spec = {
    display_name = "Test Workspace"
  }
}

resource "meshstack_location" "test" {
  metadata = {
    name               = "test-loc-[[.suffix]]"
    owned_by_workspace = meshstack_workspace.test.metadata.name
  }

  spec = {
    display_name = "My Cloud Location"
    description  = "A location for managing cloud resources"
  }
}
