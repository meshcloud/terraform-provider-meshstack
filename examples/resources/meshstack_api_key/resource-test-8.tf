# The unprivileged, workspace-scoped key a consumer runs its own building blocks with:
# BUILDINGBLOCK_* in its own workspace and deliberately no MANAGED_/ADM_ authority, so whether it may
# see a foreign definition's run logs hinges purely on that definition's run transparency.

resource "meshstack_api_key" "example" {
  metadata = {
    owned_by_workspace = meshstack_workspace.other.metadata.name
  }

  spec = {
    display_name = "ci-key"
    permissions  = ["BUILDINGBLOCK_SAVE", "BUILDINGBLOCK_LIST", "BUILDINGBLOCK_DELETE"]
  }
}
