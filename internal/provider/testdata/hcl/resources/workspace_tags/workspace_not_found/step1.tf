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
    workspace_identifier = "does-not-exist-workspace-[[.suffix]]"
  }
  spec = {
    tags = {
      (meshstack_tag_definition.test.spec.key) = ["12345"]
    }
  }
}
