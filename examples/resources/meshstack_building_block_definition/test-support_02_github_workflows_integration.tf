resource "meshstack_integration" "github" {
  metadata = {
    owned_by_workspace = "my-workspace"
  }

  spec = {
    display_name = "GitHub Integration"
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
