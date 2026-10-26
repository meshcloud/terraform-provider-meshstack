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

  # 11_run_transparency_failed_run relies on this to fail an apply on a run the provider only warns about.
  lifecycle {
    postcondition {
      condition     = self.status.succeeded
      error_message = "Building block ${self.metadata.uuid} did not succeed (health status ${self.status.health_status}, action ${self.status.action}). See its run in meshPanel."
    }
  }
}
