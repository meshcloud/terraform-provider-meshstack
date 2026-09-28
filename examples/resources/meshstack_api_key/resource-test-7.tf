# The key the definition owner drives a consumer's building block with. MANAGED_BUILDINGBLOCK_SAVE is
# the capability under test: it lets the key create and update blocks consuming its own definition in
# other workspaces, and it deliberately carries no ADM_BUILDINGBLOCK_SAVE. ADM_BUILDINGBLOCK_DELETE is
# only there so the cross-workspace block can be torn down again — there is no MANAGED delete
# authority, so a platform operator can save but not delete a block it does not own.

resource "meshstack_api_key" "example" {
  metadata = {
    owned_by_workspace = meshstack_workspace.example.metadata.name
  }

  spec = {
    display_name = "ci-key"
    permissions  = ["MANAGED_BUILDINGBLOCK_SAVE", "MANAGED_BUILDINGBLOCK_LIST", "ADM_BUILDINGBLOCK_DELETE"]
  }
}
