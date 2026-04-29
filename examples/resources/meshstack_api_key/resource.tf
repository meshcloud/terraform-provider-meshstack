resource "meshstack_api_key" "example" {
  workspace_identifier = "my-workspace"
  name                 = "ci-key"
  authorities          = ["meshfed.meshLandingZone.list", "meshfed.meshProject.list"]
  expiry_date          = "2025-12-31"
}

# The token is only available after creation.
# It is stored (sensitive) in state and cannot be retrieved again from the API.
output "api_key_token" {
  value     = meshstack_api_key.example.token
  sensitive = true
}
