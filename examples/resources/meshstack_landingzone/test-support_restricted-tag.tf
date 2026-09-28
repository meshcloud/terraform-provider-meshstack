# A restricted meshLandingZone tag definition with a default value. The backend injects that default
# into every landing zone created while it exists, whether or not the caller declares the tag — which
# is global to the kind, so the cases using this file run under TouchesExclusively.
#
# Composed alone, never beside another restricted definition: a second one would inject a second
# default and break the exact-map assertions.

resource "meshstack_tag_definition" "injected_restricted_tag" {
  spec = {
    target_kind  = "meshLandingZone"
    key          = "test-key-lz-injected-${var.suffix}"
    display_name = "Restricted Test Tag"
    restricted   = true

    value_type = {
      string = {
        default_value = "injected-default"
      }
    }
  }
}
