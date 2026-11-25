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
  workspace = "does-not-exist-workspace-[[.suffix]]"
  tags = {
    (meshstack_tag_definition.test.spec.key) = ["12345"]
  }
}
