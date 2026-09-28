# A second github integration, so a definition can switch its integration_ref away from the first
# one. Its display name differs so the backend treats it as a distinct integration.

resource "meshstack_integration" "github_b" {
  metadata = {
    owned_by_workspace = meshstack_workspace.example.metadata.name
  }

  spec = {
    display_name = "GitHub Integration B"
    config = {
      github = {
        owner           = "my-org"
        base_url        = "https://github.com"
        app_id          = "123456"
        app_private_key = { secret_value = "-----BEGIN RSA PRIVATE KEY-----\nMOCK_KEY_CONTENT\n-----END RSA PRIVATE KEY-----" }
      }
    }
  }
}
