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

resource "meshstack_tag_definition" "first" {
  spec = {
    target_kind  = "meshProject"
    key          = "test-key-first-[[.suffix]]"
    display_name = "First Test Tag"
    value_type = {
      string = {}
    }
  }
}

resource "meshstack_project_tag" "first" {
  metadata = {
    workspace_identifier = meshstack_project.test.metadata.owned_by_workspace
    project_identifier   = meshstack_project.test.metadata.name
    key                  = meshstack_tag_definition.first.spec.key
  }
  spec = {
    values = ["12345"]
  }
}

resource "meshstack_tag_definition" "second" {
  spec = {
    target_kind  = "meshProject"
    key          = "test-key-second-[[.suffix]]"
    display_name = "Second Test Tag"
    value_type = {
      string = {}
    }
  }
}

resource "meshstack_project_tag" "second" {
  metadata = {
    workspace_identifier = meshstack_project_tag.first.metadata.workspace_identifier
    project_identifier   = meshstack_project.test.metadata.name
    key                  = meshstack_tag_definition.second.spec.key
  }
  spec = {
    values = ["second"]
  }
}
