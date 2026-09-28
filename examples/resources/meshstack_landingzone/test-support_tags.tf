# The plain meshLandingZone tag definition the declared-tag cases write under.

resource "meshstack_tag_definition" "landingzone_tag" {
  spec = {
    target_kind  = "meshLandingZone"
    key          = "test-key-lz-${var.suffix}"
    display_name = "Test Tag"

    value_type = {
      string = {}
    }
  }
}
