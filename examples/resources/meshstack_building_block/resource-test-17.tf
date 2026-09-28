# The consumer's own block on a definition whose implementation is deliberately broken, created
# through the meshstack-other provider alias backed by the consumer's workspace-scoped key. The run
# fails, which the provider only warns about — the postcondition below is what turns that into a
# failed apply, and it is the recipe the resource documentation recommends.

resource "meshstack_building_block" "sensitive_user_input" {
  provider = meshstack-other

  spec = {
    building_block_definition_version_ref = { uuid = meshstack_building_block_definition.sensitive_user_input.version_latest_release.uuid }

    display_name = "my-sensitive-user-input-bb"
    target_ref   = meshstack_workspace.other.ref

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

  # The other provider needs the key for teardown, so the block has to go first.
  depends_on = [meshstack_api_key.example]

  # Bounded waits so tests fail reasonably fast (vs the 30m default) while tolerating a busy runner.
  timeouts = {
    create = "2m"
    update = "2m"
    delete = "2m"
  }

  lifecycle {
    postcondition {
      condition     = self.status.status == "SUCCEEDED"
      error_message = "Building block ${self.metadata.uuid} is ${self.status.status}, not SUCCEEDED. See its run in meshPanel."
    }
  }
}
