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

resource "meshstack_workspace" "example" {
  metadata = {
    name = "test-ws-[[.suffix]]"
    tags = {
      "cost-center-[[.suffix]]" = ["12345"]
    }
  }
  spec = {
    display_name = "Updated Display Name"
  }
}
