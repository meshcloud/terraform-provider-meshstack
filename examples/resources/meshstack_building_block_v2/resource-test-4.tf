resource "meshstack_building_block_v2" "sensitive_user_input" {
  spec = {
    building_block_definition_version_ref = meshstack_building_block_definition.sensitive_user_input.version_latest
    display_name                          = "my-sensitive-user-input-building-block"
    target_ref                            = meshstack_workspace.example.ref
    inputs = {
      secret_str  = { value_string_sensitive = "super-secret-string-value" }
      secret_code = { value_code_sensitive = "super-secret-code-value" }
    }
  }
}
