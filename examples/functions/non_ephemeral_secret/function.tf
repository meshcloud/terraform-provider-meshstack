locals {
  # A sensitive value, in practice from a sensitive variable or a secret manager.
  some_sensitive_value = sensitive("mock-pat-token-12345")
}

resource "meshstack_integration" "example" {
  metadata = {
    owned_by_workspace = "my-workspace"
  }

  spec = {
    display_name = "Azure DevOps Integration"
    config = {
      azuredevops = {
        base_url     = "https://dev.azure.com"
        organization = "my-organization"

        # Wrap the sensitive value in nonsensitive() so secret_version stays a visible hash instead
        # of "(sensitive value)". That is safe because secret_value is write only and never reaches
        # state. Prefer an ephemeral resource or keyless auth where practical.
        personal_access_token = provider::meshstack::non_ephemeral_secret(nonsensitive(local.some_sensitive_value))
      }
    }
  }
}
