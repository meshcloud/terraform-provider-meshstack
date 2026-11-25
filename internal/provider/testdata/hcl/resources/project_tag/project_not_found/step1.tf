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
    target_kind  = "meshProject"
    key          = "test-key-[[.suffix]]"
    display_name = "Test Tag"
    value_type = {
      string = {}
    }
  }
}

resource "meshstack_project_tag" "test" {
  project_ref = {
    owned_by_workspace = meshstack_workspace.test.metadata.name
    name               = "does-not-exist-project-[[.suffix]]"
  }
  key    = meshstack_tag_definition.test.spec.key
  values = ["12345"]
}
