resource "meshstack_building_block" "example_tenant" {
  spec = {
    # Alternatively, use version_latest_release to target only released versions
    building_block_definition_version_ref = one(data.meshstack_building_block_definitions.example.building_block_definitions).version_latest

    display_name = "my-tenant-building-block"
    target_ref   = one(data.meshstack_tenants.example.tenants).ref

    inputs = {
      name = {
        value = jsonencode("my-name")
      }
      size = {
        value = jsonencode(16)
      }
      environment = {
        value = jsonencode("dev")
      }
      # Sensitive inputs are supplied under `sensitive` and never stored in plaintext: the value is
      # encrypted to the runner's public key and only its sha256 hash is surfaced in `all_inputs`.
      api_key = {
        sensitive = {
          secret_value = "super-secret-api-key"
        }
      }
    }
  }

  # create/update wait for the building block run to reach a terminal state; delete waits for
  # deprovisioning. Tune to your runner's typical run duration (defaults to 30m if unset).
  timeouts = {
    create = "30s"
    update = "30s"
    delete = "30s"
  }
}
