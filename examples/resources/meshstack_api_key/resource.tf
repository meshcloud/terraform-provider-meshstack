resource "meshstack_api_key" "example" {
  workspace_identifier = "my-workspace"
  display_name         = "ci-key"
  authorities          = ["LANDINGZONE_LIST", "PROJECT_LIST"]
  expires_at           = "2025-12-31"
}

# The token is only available after creation or after secret rotation (expires_at change).
# It is stored (sensitive) in state and cannot be retrieved again from the API.
output "api_key_token" {
  value     = meshstack_api_key.example.token
  sensitive = true
}
