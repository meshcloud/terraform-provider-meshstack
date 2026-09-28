# The workspace tag that the building block definition reads as a TAG input.
resource "meshstack_tag_definition" "workspace_business_unit" {
  spec = {
    target_kind  = "meshWorkspace"
    key          = "business-unit-${var.tag_suffix}"
    display_name = "Business Unit"

    value_type = {
      string = {}
    }
  }
}
