resource "meshstack_building_block_v2" "sensitive_user_input" {
  spec = {
    building_block_definition_version_ref = { uuid = "placeholder" }
    display_name                          = "my-sensitive-user-input-building-block"
    target_ref                            = { kind = "meshWorkspace", name = "placeholder" }
    inputs = {
      secret_str  = { value_string_sensitive = "super-secret-string-value" }
      secret_code = { value_code_sensitive = "super-secret-code-value" }
    }
  }
  wait_for_completion = false
  purge_on_delete     = true
}
