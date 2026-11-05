resource "meshstack_tag_definition" "declared" {
  spec = {
    target_kind  = "meshWorkspace"
    key          = "test-key-[[.suffix]]"
    display_name = "Test Tag"
    value_type = {
      string = {}
    }
  }
}

resource "meshstack_tag_definition" "undeclared" {
  spec = {
    target_kind  = "meshWorkspace"
    key          = "test-undeclared-key-[[.suffix]]"
    display_name = "Undeclared Test Tag"
    value_type = {
      string = {}
    }
  }
}

resource "meshstack_workspace" "test" {
  metadata = {
    name = "test-ws-[[.suffix]]"
    tags = {
      (meshstack_tag_definition.declared.spec.key) = ["12345"]
    }
  }
  spec = {
    display_name = "Test Workspace"
  }
}
