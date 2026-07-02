resource "meshstack_building_block" "sensitive_user_input" {
  spec = {
    building_block_definition_version_ref = { uuid = "placeholder" }
    display_name                          = "my-sensitive-user-input-bb"
    target_ref                            = { kind = "meshWorkspace", name = "placeholder" }

    inputs = {
      api_key = {
        sensitive = {
          secret_value = "super-secret-api-key"
        }
      }
      script = {
        sensitive = {
          secret_value = "#!/bin/bash\necho super-secret-script"
        }
      }
    }
  }

  # Short waits so tests fail fast instead of polling the 30m default if a run hangs.
  timeouts = {
    create = "30s"
    update = "30s"
    delete = "30s"
  }
}
