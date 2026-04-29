resource "meshstack_api_key" "example" {
  metadata = {
    owned_by_workspace = data.meshstack_workspace.example.metadata.name
  }

  spec = {
    display_name = "ci-key"
    permissions  = ["LANDINGZONE_LIST", "PROJECT_LIST"]
    # expires_at is optional; if omitted the key never expires.
    # Setting an expiry is recommended for security best practices.
    # expires_at = "2025-12-31"
  }
}
