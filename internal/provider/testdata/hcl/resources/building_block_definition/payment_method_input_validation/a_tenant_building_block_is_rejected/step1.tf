resource "meshstack_building_block_definition" "test" {
  metadata = { owned_by_workspace = "test-ws-[[.suffix]]" }
  spec = {
    display_name        = "Test"
    description         = "Test"
    target_type         = "TENANT_LEVEL"
    supported_platforms = [{ name = "AZURE" }]
  }
  version_spec = {
    draft          = true
    implementation = { terraform = { terraform_version = "1.9.0", repository_url = "https://example.com/bb.git" } }
    inputs = {
      payment_method = {
        display_name    = "Payment Method"
        assignment_type = "PAYMENT_METHOD"
        type            = "CODE"
      }
    }
  }
}
