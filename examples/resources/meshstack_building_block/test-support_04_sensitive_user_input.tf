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

  # Bounded waits so tests fail reasonably fast (vs the 30m default) while tolerating a busy runner.
  timeouts = {
    create = "2m"
    update = "2m"
    delete = "2m"
  }
}
