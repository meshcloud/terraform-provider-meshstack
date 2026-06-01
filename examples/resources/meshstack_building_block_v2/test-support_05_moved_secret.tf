resource "meshstack_building_block_v2" "moved_secret" {
  spec = {
    building_block_definition_version_ref = { uuid = "placeholder" }
    display_name                          = "my-moved-secret-building-block"
    target_ref                            = { kind = "meshWorkspace", name = "placeholder" }
    inputs = {
      api_key = { value_string_sensitive = "super-secret-api-key" }
      script  = { value_code_sensitive = "#!/bin/bash\necho super-secret-script" }
    }
  }
}
