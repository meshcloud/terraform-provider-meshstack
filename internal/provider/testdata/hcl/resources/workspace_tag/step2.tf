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

[[ example "resources/meshstack_workspace_tag/resource.tf" ]]
