# The v3 side of the v2->v3 migration of a block with sensitive inputs. The real secret cannot ride
# through state, because secret_value is write-only. Re-declaring it with a DISTINCT placeholder and
# no version preserves the stored secret — the provider echoes back the hash it recovered on refresh
# — and the placeholder is what makes corruption detectable: sending it would change the hash.

resource "meshstack_building_block" "moved_secret" {
  spec = {
    building_block_definition_version_ref = meshstack_building_block_definition.sensitive_user_input.version_latest

    display_name = "my-moved-secret-building-block"
    target_ref   = meshstack_workspace.example.ref

    inputs = {
      api_key = {
        sensitive = {
          secret_value   = var.moved_secret_value
          secret_version = var.moved_secret_version
        }
      }
      script = {
        sensitive = {
          secret_value = "placeholder-not-the-real-script"
        }
      }
    }
  }
}
