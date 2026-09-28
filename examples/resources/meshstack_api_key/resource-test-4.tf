# The restricted key the consumer workspace reads with: list permission only, so what it can see is
# decided by entitlement rather than by scope.

resource "meshstack_api_key" "example" {
  metadata = {
    owned_by_workspace = meshstack_workspace.other.metadata.name
  }

  spec = {
    display_name = "ci-key"
    permissions  = ["PLATFORMINSTANCE_LIST"]
  }
}
