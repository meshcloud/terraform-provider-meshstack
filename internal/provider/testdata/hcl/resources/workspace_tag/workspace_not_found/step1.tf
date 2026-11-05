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

resource "meshstack_workspace_tag" "test" {
  metadata = {
    workspace_identifier = "does-not-exist-workspace-[[.suffix]]"
    key                  = meshstack_tag_definition.test.spec.key
  }
  spec = {
    values = ["12345"]
  }
}
