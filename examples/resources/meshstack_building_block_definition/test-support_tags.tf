# The meshBuildingBlockDefinition tag the definition declares, and a restricted one whose default the
# backend injects into every definition created while it exists — which is global to the kind, so the
# case using this file runs under TouchesExclusively.

resource "meshstack_tag_definition" "bbd_tag" {
  spec = {
    target_kind  = "meshBuildingBlockDefinition"
    key          = "test-key-bbd-${var.tag_suffix}"
    display_name = "Test Tag"

    value_type = {
      string = {}
    }
  }
}

resource "meshstack_tag_definition" "bbd_restricted_tag" {
  spec = {
    target_kind  = "meshBuildingBlockDefinition"
    key          = "test-key-bbd-restricted-${var.tag_suffix}"
    display_name = "Restricted Test Tag"
    restricted   = true

    value_type = {
      string = {
        default_value = "injected-default"
      }
    }
  }
}
