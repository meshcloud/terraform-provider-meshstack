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
  workspace = meshstack_workspace.test.metadata.name
  key       = meshstack_tag_definition.a.spec.key
  values    = ["a"]
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
  workspace = meshstack_workspace.test.metadata.name
  key       = meshstack_tag_definition.b.spec.key
  values    = ["b"]
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
  workspace = meshstack_workspace.test.metadata.name
  key       = meshstack_tag_definition.c.spec.key
  values    = ["c"]
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
  workspace = meshstack_workspace.test.metadata.name
  key       = meshstack_tag_definition.d.spec.key
  values    = ["d"]
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
  workspace = meshstack_workspace.test.metadata.name
  key       = meshstack_tag_definition.e.spec.key
  values    = ["e"]
}
