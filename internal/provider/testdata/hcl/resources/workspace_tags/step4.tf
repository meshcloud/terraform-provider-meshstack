resource "meshstack_workspace" "test" {
  metadata = {
    name = "test-ws-[[.suffix]]"
  }
  spec = {
    display_name = "Test Workspace"
  }
}

data "meshstack_workspace" "example" {
  metadata = {
    name = meshstack_workspace.test.metadata.name
  }
}

resource "meshstack_tag_definition" "cost_center" {
  spec = {
    target_kind  = "meshWorkspace"
    key          = "cost-center-[[.suffix]]"
    display_name = "Cost Center"
    value_type = {
      string = {}
    }
  }
}

resource "meshstack_workspace_tags" "example" {
  metadata = {
    workspace_identifier = data.meshstack_workspace.example.metadata.name
  }
  spec = {
    tags = {
      "cost-center-[[.suffix]]" = ["12345"]
    }
  }
}
