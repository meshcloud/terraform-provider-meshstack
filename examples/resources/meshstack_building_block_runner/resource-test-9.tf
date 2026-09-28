# The published workload identity federation example (resource_wif.tf), whose runner declares a subject
# template.

resource "meshstack_building_block_runner" "example_with_wif" {
  metadata = {
    owned_by_workspace = meshstack_workspace.example.metadata.name
  }

  spec = {
    display_name        = "My GCP WIF Runner"
    implementation_type = "TERRAFORM"
    public_key          = var.runner_public_key
    restriction         = "PRIVATE"

    workload_identity_federation = {
      subject_template = var.subject_template
      issuer           = "https://oidc.example.com"

      gcp = {
        audience   = "gcp-workload-identity-provider:namespace"
        token_path = "/var/run/secrets/workload-identity/token"
      }
    }
  }
}
