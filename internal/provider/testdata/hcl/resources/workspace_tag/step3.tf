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
      multi_select = {
        options = ["12345", "67890"]
      }
    }
  }
}

resource "meshstack_workspace_tag" "example" {
  workspace = data.meshstack_workspace.example.metadata.name
  key       = meshstack_tag_definition.cost_center.spec.key
  values    = ["12345", "67890"]
}
