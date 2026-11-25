resource "meshstack_workspace" "test" {
  metadata = {
    name = "test-ws-[[.suffix]]"
  }
  spec = {
    display_name = "Test Workspace"
  }
}

resource "meshstack_tag_definition" "first" {
  spec = {
    target_kind  = "meshWorkspace"
    key          = "test-key-first-[[.suffix]]"
    display_name = "First Test Tag"
    value_type = {
      string = {}
    }
  }
}

resource "meshstack_workspace_tag" "first" {
  workspace = meshstack_workspace.test.metadata.name
  key       = meshstack_tag_definition.first.spec.key
  values    = ["12345"]
}

resource "meshstack_tag_definition" "second" {
  spec = {
    target_kind  = "meshWorkspace"
    key          = "test-key-second-[[.suffix]]"
    display_name = "Second Test Tag"
    value_type = {
      string = {}
    }
  }
}

resource "meshstack_workspace_tag" "second" {
  workspace = meshstack_workspace_tag.first.workspace
  key       = meshstack_tag_definition.second.spec.key
  values    = ["second"]
}
