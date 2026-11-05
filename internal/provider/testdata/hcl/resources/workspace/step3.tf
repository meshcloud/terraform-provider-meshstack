resource "meshstack_tag_definition" "cost_center" {
  spec = {
    target_kind  = "meshWorkspace"
    key          = "cost-center"
    display_name = "Cost Center"
    value_type = {
      string = {}
    }
  }
}

resource "meshstack_workspace" "example" {
  metadata = {
    name = "my-new-workspace"
    tags = {
      "cost-center" = ["12345"]
    }
  }
  spec = {
    display_name = "Updated Display Name"
  }
}
