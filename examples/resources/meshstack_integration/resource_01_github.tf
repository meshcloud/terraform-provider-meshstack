resource "meshstack_integration" "example_github" {
  metadata = {
    owned_by_workspace = data.meshstack_workspace.example.metadata.name
  }

  spec = {
    display_name = "GitHub Integration"
    config = {
      github = {
        owner           = "my-org"
        base_url        = "https://github.com"
        app_id          = "123456"
        app_private_key = { secret_value = "-----BEGIN RSA PRIVATE KEY-----\nMOCK_KEY_CONTENT\n-----END RSA PRIVATE KEY-----" }
        runner_ref      = { uuid = "98520496-627d-43e6-82da-ce499179ff3f" } # Optional, by default, pre-defined shared runner is used
      }
    }
  }
}
