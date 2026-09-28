# The v2 side of the v2->v3 migration of a block with sensitive inputs. It supplies the real secrets;
# the v3 step file that replaces it re-declares them with placeholders and must not overwrite them.

resource "meshstack_building_block_v2" "moved_secret" {
  spec = {
    building_block_definition_version_ref = meshstack_building_block_definition.sensitive_user_input.version_latest

    display_name = "my-moved-secret-building-block"
    target_ref   = meshstack_workspace.example.ref

    inputs = {
      api_key = { value_string_sensitive = "super-secret-api-key" }
      script  = { value_code_sensitive = "#!/bin/bash\necho super-secret-script" }
    }
  }
}
