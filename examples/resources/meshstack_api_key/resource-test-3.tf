resource "meshstack_api_key" "example" {
  metadata = {
    owned_by_workspace = meshstack_workspace.example.metadata.name
  }

  spec = {
    display_name = "ci-key"
    permissions  = ["LANDINGZONE_LIST", "PROJECT_LIST"]
    # Setting an expiry rotates the key, so the secret is unknown at plan time.
    expires_at = "2099-06-30"
  }
}
