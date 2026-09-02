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

  # The strict form of the postcondition the resource example carries. Every run of this definition is
  # expected to reach SUCCEEDED, so anything else is a test failure — which is what
  # 11_run_transparency_failed_run relies on to fail an apply on a run the provider only warns about.
  lifecycle {
    postcondition {
      condition     = self.status.status == "SUCCEEDED"
      error_message = "Building block ${self.metadata.uuid} is ${self.status.status}, not SUCCEEDED. See its run in meshPanel."
    }
  }
}
