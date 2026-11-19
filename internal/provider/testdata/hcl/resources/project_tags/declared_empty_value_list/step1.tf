resource "meshstack_workspace" "test" {
  metadata = {
    name = "test-ws-[[.suffix]]"
  }
  spec = {
    display_name = "Test Workspace"
  }
}

resource "meshstack_project" "test" {
  metadata = {
    name               = "test-project"
    owned_by_workspace = meshstack_workspace.test.metadata.name
  }
  spec = {
    display_name = "Test Project"
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

resource "meshstack_project_tags" "test" {
  metadata = {
    workspace_identifier = meshstack_project.test.metadata.owned_by_workspace
    project_identifier   = meshstack_project.test.metadata.name
  }
  spec = {
    tags = {
      (meshstack_tag_definition.test.spec.key) = []
    }
  }
}
