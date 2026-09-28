# A block with sensitive user inputs, pinned to its definition's latest *released* version so it
# stays put while a draft exists and upgrades when that draft is released. api_key declares a
# secret_version: the backend returns a hash rather than the value, so bumping the version is the
# only way a rotation becomes visible to the provider.

resource "meshstack_building_block" "sensitive_user_input" {
  spec = {
    building_block_definition_version_ref = { uuid = meshstack_building_block_definition.sensitive_user_input.version_latest_release.uuid }

    display_name = "my-sensitive-user-input-bb"
    target_ref   = meshstack_workspace.example.ref

    inputs = {
      api_key = {
        sensitive = {
          secret_value   = var.secret_value
          secret_version = var.secret_version
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
  # expected to reach SUCCEEDED, so anything else is a test failure.
  lifecycle {
    postcondition {
      condition     = self.status.status == "SUCCEEDED"
      error_message = "Building block ${self.metadata.uuid} is ${self.status.status}, not SUCCEEDED. See its run in meshPanel."
    }
  }
}
