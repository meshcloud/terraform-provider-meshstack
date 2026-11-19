resource "meshstack_workspace" "test" {
  metadata = {
    name = "test-ws-[[.suffix]]"
  }
  spec = {
    display_name = "Test Workspace"
  }
}

resource "meshstack_tag_definition" "a" {
  spec = {
    target_kind  = "meshWorkspace"
    key          = "test-key-a-[[.suffix]]"
    display_name = "Parallel Test Tag a"
    value_type = {
      string = {}
    }
  }
}

resource "meshstack_workspace_tag" "a" {
  metadata = {
    workspace_identifier = meshstack_workspace.test.metadata.name
    key                  = meshstack_tag_definition.a.spec.key
  }
  spec = {
    values = ["a"]
  }
}

resource "meshstack_tag_definition" "b" {
  spec = {
    target_kind  = "meshWorkspace"
    key          = "test-key-b-[[.suffix]]"
    display_name = "Parallel Test Tag b"
    value_type = {
      string = {}
    }
  }
}

resource "meshstack_workspace_tag" "b" {
  metadata = {
    workspace_identifier = meshstack_workspace.test.metadata.name
    key                  = meshstack_tag_definition.b.spec.key
  }
  spec = {
    values = ["b"]
  }
}

resource "meshstack_tag_definition" "c" {
  spec = {
    target_kind  = "meshWorkspace"
    key          = "test-key-c-[[.suffix]]"
    display_name = "Parallel Test Tag c"
    value_type = {
      string = {}
    }
  }
}

resource "meshstack_workspace_tag" "c" {
  metadata = {
    workspace_identifier = meshstack_workspace.test.metadata.name
    key                  = meshstack_tag_definition.c.spec.key
  }
  spec = {
    values = ["c"]
  }
}

resource "meshstack_tag_definition" "d" {
  spec = {
    target_kind  = "meshWorkspace"
    key          = "test-key-d-[[.suffix]]"
    display_name = "Parallel Test Tag d"
    value_type = {
      string = {}
    }
  }
}

resource "meshstack_workspace_tag" "d" {
  metadata = {
    workspace_identifier = meshstack_workspace.test.metadata.name
    key                  = meshstack_tag_definition.d.spec.key
  }
  spec = {
    values = ["d"]
  }
}

resource "meshstack_tag_definition" "e" {
  spec = {
    target_kind  = "meshWorkspace"
    key          = "test-key-e-[[.suffix]]"
    display_name = "Parallel Test Tag e"
    value_type = {
      string = {}
    }
  }
}

resource "meshstack_workspace_tag" "e" {
  metadata = {
    workspace_identifier = meshstack_workspace.test.metadata.name
    key                  = meshstack_tag_definition.e.spec.key
  }
  spec = {
    values = ["e"]
  }
}
