resource "meshstack_integration" "example_azure_devops" {
  metadata = {
    owned_by_workspace = data.meshstack_workspace.example.metadata.name
  }

  spec = {
    display_name = "Azure DevOps Integration"
    config = {
      azuredevops = {
        base_url     = "https://dev.azure.com"
        organization = "my-organization"

        # non_ephemeral_secret ties rotation to the value's hash. Change the value and secret_version
        # changes with it, which sends the write only secret_value again. An unchanged value produces
        # no diff. Wrap a sensitive input in nonsensitive() to keep the version hash visible in plans.
        # See the non_ephemeral_secret function docs.
        personal_access_token = provider::meshstack::non_ephemeral_secret("mock-pat-token-12345")
      }
    }
  }
}
