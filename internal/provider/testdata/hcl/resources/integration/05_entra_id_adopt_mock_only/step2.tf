data "meshstack_instance" "test" {}

resource "meshstack_integration" "test" {
  metadata = {
    owned_by_workspace = data.meshstack_instance.test.admin_workspace_identifier
  }

  spec = {
    display_name = "Entra ID Integration"
    config = {
      entraid = {
        tenant_id     = "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
        client_id     = "yyyyyyyy-yyyy-yyyy-yyyy-yyyyyyyyyyyy"
        client_secret = { secret_value = "my-client-secret" }
      }
    }
  }
}
