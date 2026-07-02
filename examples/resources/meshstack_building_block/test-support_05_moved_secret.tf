resource "meshstack_building_block" "moved_secret" {
  spec = {
    building_block_definition_version_ref = { uuid = "placeholder" }
    display_name                          = "my-moved-secret-building-block"
    target_ref                            = { kind = "meshWorkspace", name = "placeholder" }

    inputs = {
      # After the move the real secret cannot ride through state (secret_value is write-only).
      # moveFromV2 recovers the secret's hash on refresh, so re-declaring with a DISTINCT placeholder
      # secret_value and NO secret_version preserves the existing secret (the provider echoes the
      # stored hash) — the backend keeps the original value and the hash stays equal to the v2
      # block's. A distinct placeholder (not the real value) is what makes corruption detectable.
      api_key = {
        sensitive = {
          secret_value = "placeholder-not-the-real-api-key"
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
