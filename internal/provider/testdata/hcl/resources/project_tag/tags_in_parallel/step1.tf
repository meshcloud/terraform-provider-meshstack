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

resource "meshstack_tag_definition" "a" {
  spec = {
    target_kind  = "meshProject"
    key          = "test-key-a-[[.suffix]]"
    display_name = "Parallel Test Tag a"
    value_type = {
      string = {}
    }
  }
}

resource "meshstack_project_tag" "a" {
  project_ref = {
    owned_by_workspace = meshstack_project.test.metadata.owned_by_workspace
    name               = meshstack_project.test.metadata.name
  }
  key    = meshstack_tag_definition.a.spec.key
  values = ["a"]
}

resource "meshstack_tag_definition" "b" {
  spec = {
    target_kind  = "meshProject"
    key          = "test-key-b-[[.suffix]]"
    display_name = "Parallel Test Tag b"
    value_type = {
      string = {}
    }
  }
}

resource "meshstack_project_tag" "b" {
  project_ref = {
    owned_by_workspace = meshstack_project.test.metadata.owned_by_workspace
    name               = meshstack_project.test.metadata.name
  }
  key    = meshstack_tag_definition.b.spec.key
  values = ["b"]
}

resource "meshstack_tag_definition" "c" {
  spec = {
    target_kind  = "meshProject"
    key          = "test-key-c-[[.suffix]]"
    display_name = "Parallel Test Tag c"
    value_type = {
      string = {}
    }
  }
}

resource "meshstack_project_tag" "c" {
  project_ref = {
    owned_by_workspace = meshstack_project.test.metadata.owned_by_workspace
    name               = meshstack_project.test.metadata.name
  }
  key    = meshstack_tag_definition.c.spec.key
  values = ["c"]
}

resource "meshstack_tag_definition" "d" {
  spec = {
    target_kind  = "meshProject"
    key          = "test-key-d-[[.suffix]]"
    display_name = "Parallel Test Tag d"
    value_type = {
      string = {}
    }
  }
}

resource "meshstack_project_tag" "d" {
  project_ref = {
    owned_by_workspace = meshstack_project.test.metadata.owned_by_workspace
    name               = meshstack_project.test.metadata.name
  }
  key    = meshstack_tag_definition.d.spec.key
  values = ["d"]
}

resource "meshstack_tag_definition" "e" {
  spec = {
    target_kind  = "meshProject"
    key          = "test-key-e-[[.suffix]]"
    display_name = "Parallel Test Tag e"
    value_type = {
      string = {}
    }
  }
}

resource "meshstack_project_tag" "e" {
  project_ref = {
    owned_by_workspace = meshstack_project.test.metadata.owned_by_workspace
    name               = meshstack_project.test.metadata.name
  }
  key    = meshstack_tag_definition.e.spec.key
  values = ["e"]
}
