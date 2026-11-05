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

resource "meshstack_tag_definition" "budget_code" {
  spec = {
    target_kind  = "meshWorkspace"
    key          = "budget-code"
    display_name = "Budget Code"
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
      "budget-code" = ["12345"]
    }
  }
}
