resource "meshstack_building_block_runner" "example" {
  metadata = {
    owned_by_workspace = data.meshstack_workspace.example.metadata.name
  }

  spec = {
    display_name        = "My Terraform Runner"
    implementation_type = "TERRAFORM"
    public_key          = "-----BEGIN PUBLIC KEY-----\nMIICIjANBgkqhkiG9w0BAQEFAAOCAg8AMIICCgKCAgEA...\n-----END PUBLIC KEY-----"
    restriction         = "PRIVATE"
  }
}
