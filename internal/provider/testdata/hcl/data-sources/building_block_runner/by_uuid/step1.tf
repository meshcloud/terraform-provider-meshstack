resource "meshstack_workspace" "test" {
  metadata = {
    name = "test-ws-[[.suffix]]"
  }
  spec = {
    display_name = "Test Workspace"
  }
}

data "meshstack_workspace" "example" {
  metadata = {
    name = meshstack_workspace.test.metadata.name
  }
}

resource "meshstack_building_block_runner" "example_with_wif" {
  metadata = {
    owned_by_workspace = data.meshstack_workspace.example.metadata.name
  }

  spec = {
    display_name        = "My GCP WIF Runner"
    implementation_type = "TERRAFORM"
    public_key          = <<-EOT
      -----BEGIN PUBLIC KEY-----
      MIIBIjANBgkqhkiG9w0BAQEFAAOCAQ8AMIIBCgKCAQEAuP1aaqG5oCtnyPUWVbor
      11jsYaOt5A/IzOCA8QsrKFIpMlPOTl9HNsKDDUmYPMNI6uNCpFHmcV46/cqr47s8
      s8NwxckJeHLDItGXGJ8Z2k2YUSp4UPH0PxTNqr5XDIzucxYEoRPluy3jin54B6qU
      gd/BSLJq5e9Ifx00G24FMqJ1Bc24/lYVnBxuXkFHGXJMpfVTgOPlEbIjHYeSfB6U
      QVjUsLddpJCKG/dtQYLrufH3k7A8d6G3PlMe+BNOJpNc9zSxzd6bgbY50Q8BHnmI
      D6/hNv0bNwwerI9eVDrsyDzQ/5YiK+APjAHrzqpCizooYyHgzvqN6aHQEGm4EC0w
      VwIDAQAB
      -----END PUBLIC KEY-----
    EOT
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

data "meshstack_building_block_runner" "default" {
  metadata = {
    uuid = meshstack_building_block_runner.example_with_wif.metadata.uuid
  }
}
