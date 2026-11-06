resource "meshstack_workspace" "test" {
  metadata = {
    name = "test-ws-[[.suffix]]"
  }
  spec = {
    display_name = "Test Workspace"
  }
}

data "meshstack_workspace" "example" {
  metadata = {
    name = meshstack_workspace.test.metadata.name
  }
}

resource "meshstack_workspace" "other" {
  metadata = {
    name = "test-ws-other-[[.suffix]]"
  }
  spec = {
    display_name = "Other Test Workspace"
  }
}

resource "meshstack_building_block_definition" "sensitive_user_input" {
  metadata = {
    owned_by_workspace = data.meshstack_workspace.example.metadata.name
  }

  spec = {
    display_name     = "BB v3 Sensitive Inputs Test Definition"
    description      = "Definition covering STRING/CODE/STATIC sensitive inputs for the BB v3 sensitive+upgrade scenario"
    run_transparency = [[.run_transparency]]
  }

  version_spec = {
    draft = false

    inputs = {
      # STRING-typed sensitive USER_INPUT. The consumer supplies it; the backend (and the mock)
      # surface its sha256 hash in all_inputs. Also the input that gets rotated and carried
      # through the v1->v2 upgrade in the acceptance-only steps.
      api_key = {
        display_name    = "API Key"
        type            = "STRING"
        assignment_type = "USER_INPUT"
        sensitive       = {}
      }
      # CODE-typed sensitive USER_INPUT. It takes the identical code path as the STRING api_key — the
      # only difference is that its hash surfaces in value_code instead of value_string. Works in both
      # mock and acc (the mock's backendSecretBehavior hashes the SecretEmbedded plaintext).
      script = {
        display_name    = "Startup Script"
        type            = "CODE"
        assignment_type = "USER_INPUT"
        sensitive       = {}
      }
      # STATIC sensitive input. The consumer never supplies it; the backend resolves it from this
      # argument and surfaces its hash in all_inputs. The mock does not resolve STATIC inputs, so
      # this only appears against the real backend (asserted acceptance-only).
      static_secret = {
        display_name    = "Static Secret"
        type            = "STRING"
        assignment_type = "STATIC"
        sensitive = {
          argument = {
            secret_value = "super-secret-static-value"
          }
        }
      }
    }

    implementation = {
      terraform = {
        terraform_version = "1.9.0"
        repository_url    = "[[.repository_url]]"
        ref_name          = "broken"
      }
    }

    outputs       = {}
    deletion_mode = "PURGE"
  }
}

resource "meshstack_api_key" "reader" {
  metadata = {
    owned_by_workspace = meshstack_workspace.other.metadata.name
  }

  spec = {
    display_name = "ci-key"
    permissions  = ["BUILDINGBLOCK_SAVE", "BUILDINGBLOCK_LIST", "BUILDINGBLOCK_DELETE"]
    # expires_at is optional; if omitted the key never expires.
    # Setting an expiry is recommended for security best practices.
    # expires_at = "2025-12-31"
  }
}
