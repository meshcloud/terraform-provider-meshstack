data "meshstack_tag_definition" "example" {
  metadata = {
    name = "meshWorkspace.costCenter"
  }
  # or look it up by what it tags and its key:
  # spec = {
  #   target_kind = "meshWorkspace"
  #   key         = "costCenter"
  # }
}
