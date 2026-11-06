resource "meshstack_workspace" "test" {
  metadata = {
    name = "test-ws-[[.suffix]]"
  }
  spec = {
    display_name = "Test Workspace"
  }
}

resource "meshstack_tag_definition" "test" {
  spec = {
    target_kind  = "meshWorkspace"
    key          = "test-key-[[.suffix]]"
    display_name = "Test Tag"
    value_type = {
      string = {}
    }
  }
}

resource "meshstack_workspace_tags" "test" {
  metadata = {
    workspace_identifier = meshstack_workspace.test.metadata.name
  }
  spec = {
    tags = {
      (meshstack_tag_definition.test.spec.key) = []
    }
  }
}
