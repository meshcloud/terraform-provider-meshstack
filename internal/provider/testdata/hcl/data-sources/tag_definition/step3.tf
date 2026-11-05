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
  spec = {
    target_kind = meshstack_tag_definition.cost_center.spec.target_kind
    key         = meshstack_tag_definition.cost_center.spec.key
  }
}
