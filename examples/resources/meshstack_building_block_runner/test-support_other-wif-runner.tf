# A second runner with workload identity federation in the same workspace, for a definition to move to.
# Its issuer and subject template differ from resource-test-9.tf's, so the identity a version reports
# shows which runner it runs on.
resource "meshstack_building_block_runner" "example_with_other_wif" {
  metadata = {
    owned_by_workspace = meshstack_workspace.example.metadata.name
  }

  spec = {
    display_name        = "My Other GCP WIF Runner"
    implementation_type = "TERRAFORM"
    public_key          = var.runner_public_key
    restriction         = "PRIVATE"

    workload_identity_federation = {
      subject_template = "system:serviceaccount:other-namespace:bbd.{{ buildingBlockDefinitionUuid }}"
      issuer           = "https://oidc-other.example.com"

      gcp = {
        audience   = "gcp-workload-identity-provider:namespace"
        token_path = "/var/run/secrets/workload-identity/token"
      }
    }
  }
}
