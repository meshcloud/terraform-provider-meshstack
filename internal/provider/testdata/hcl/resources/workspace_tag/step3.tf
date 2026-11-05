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

resource "meshstack_tag_definition" "billing_account" {
  spec = {
    target_kind  = "meshWorkspace"
    key          = "billing-account"
    display_name = "Billing Account"
    value_type = {
      string = {}
    }
  }
}

resource "meshstack_workspace_tag" "example" {
  metadata = {
    workspace_identifier = data.meshstack_workspace.example.metadata.name
    key                  = "billing-account"
  }
  spec = {
    values = ["12345", "67890"]
  }
}
