# A second restricted meshLandingZone tag definition, this one the landing zone declares a value for.
# Used only alongside test-support_restricted-tag.tf, to show that a declared restricted tag is
# tracked and round-trips while an undeclared injected one is still reconciled away.

resource "meshstack_tag_definition" "declared_restricted_tag" {
  spec = {
    target_kind  = "meshLandingZone"
    key          = "test-key-lz-declared-${var.suffix}"
    display_name = "Restricted Test Tag"
    restricted   = true

    value_type = {
      string = {
        default_value = "default-value"
      }
    }
  }
}
