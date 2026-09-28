resource "meshstack_building_block_runner" "example" {
  metadata = {
    owned_by_workspace = meshstack_workspace.example.metadata.name
  }

  spec = {
    display_name        = "My Terraform Runner"
    implementation_type = "TERRAFORM"
    public_key          = var.runner_public_key
    restriction         = "PRIVATE"

    # subject_template omitted: it is required.
    workload_identity_federation = {
      issuer = "https://oidc.example.com"

      gcp = {
        audience   = "//iam.googleapis.com/projects/123456/locations/global/workloadIdentityPools/meshstack/providers/meshfed"
        token_path = "/var/run/secrets/workload-identity/token"
      }
    }
  }
}
