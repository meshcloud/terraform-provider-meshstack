resource "meshstack_building_block_runner" "example_with_wif" {
  metadata = {
    owned_by_workspace = data.meshstack_workspace.example.metadata.name
  }

  spec = {
    display_name        = "My GCP WIF Runner"
    implementation_type = "TERRAFORM"
    public_key          = "-----BEGIN PUBLIC KEY-----\nMIICIjANBgkqhkiG9w0BAQEFAAOCAg8AMIICCgKCAgEA...\n-----END PUBLIC KEY-----"
    restriction         = "PRIVATE"

    workload_identity_federation = {
      # meshStack fills in the placeholders for each building block definition and reports the result as
      # that definition's version_latest.workload_identity_federation.subject.
      subject_template = "system:serviceaccount:namespace:workspace.{{ workspaceIdentifier }}.buildingblockdefinition.{{ buildingBlockDefinitionUuid }}"
      issuer           = "https://oidc.example.com"

      gcp = {
        audience   = "gcp-workload-identity-provider:namespace"
        token_path = "/var/run/secrets/workload-identity/token"
      }
    }
  }
}
