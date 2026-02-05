resource "meshstack_integration" "example_github" {
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
        app_private_key = "-----BEGIN RSA PRIVATE KEY-----\nMOCK_KEY_CONTENT\n-----END RSA PRIVATE KEY-----"
        runner_ref = {
          uuid = "dc8c57a1-823f-4e96-8582-0275fa27dc7b"
        }
      }
    }
  }
}

# Access workload identity federation for GCP
output "github_wif_gcp_audience" {
  value = meshstack_integration.example_github.status.workload_identity_federation.gcp.audience
}
