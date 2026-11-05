resource "meshstack_tag_definition" "cost_center" {
  spec = {
    target_kind  = "meshWorkspace"
    key          = "costCenter-[[.suffix]]"
    display_name = "Cost Center"
    value_type = {
      string = {}
    }
  }
}

data "meshstack_tag_definition" "example" {
  metadata = {
    name = meshstack_tag_definition.cost_center.metadata.name
  }
  spec = {
    target_kind = "meshProject"
  }
}
