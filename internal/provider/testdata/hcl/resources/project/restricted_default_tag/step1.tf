resource "meshstack_workspace" "test" {
  metadata = {
    name = "test-ws-[[.suffix]]"
  }
  spec = {
    display_name = "Test Workspace"
  }
}

resource "meshstack_tag_definition" "tag" {
  spec = {
    target_kind  = "meshProject"
    key          = "test-key-[[.suffix]]"
    display_name = "Test Tag"
    value_type = {
      string = {}
    }
  }
}

resource "meshstack_tag_definition" "restricted_tag" {
  spec = {
    target_kind  = "meshProject"
    key          = "test-restricted-key-[[.suffix]]"
    display_name = "Restricted Test Tag"
    restricted   = true
    value_type = {
      string = {
        default_value = "injected-default"
      }
    }
  }
}

resource "meshstack_project" "test" {
  metadata = {
    name               = "test-proj-[[.suffix]]"
    owned_by_workspace = meshstack_workspace.test.metadata.name
  }
  spec = {
    display_name = "Test Project"
    tags = {
      (meshstack_tag_definition.tag.spec.key) = ["blue"]
    }
  }
  depends_on = [meshstack_tag_definition.restricted_tag]
}
