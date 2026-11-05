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

resource "meshstack_building_block_definition" "example" {
  metadata = {
    owned_by_workspace = data.meshstack_workspace.example.metadata.name
  }

  spec = {
    display_name     = "Test BB v3 Operator Inputs"
    description      = "A building block definition with user and operator inputs"
    run_transparency = true
  }

  # Reuses the same input keys as the workspace example (resource_01_workspace.tf), except its
  # `payment_method`, but changes their behaviour: `size` is a platform-operator input (settable only by
  # an operator/admin), while `name`/`environment` stay regular consumer inputs.
  version_spec = {
    draft = false

    inputs = {
      name = {
        display_name           = "Name"
        description            = "Name of the resource"
        type                   = "STRING"
        assignment_type        = "USER_INPUT"
        updateable_by_consumer = true
      }
      size = {
        display_name           = "Size"
        description            = "A platform operator input"
        type                   = "INTEGER"
        assignment_type        = "PLATFORM_OPERATOR_MANUAL_INPUT"
        updateable_by_consumer = true
      }
      environment = {
        display_name           = "Environment"
        description            = "Target environment"
        type                   = "SINGLE_SELECT"
        assignment_type        = "USER_INPUT"
        updateable_by_consumer = true
        selectable_values      = ["dev", "staging", "prod"]
      }
    }

    implementation = {
      manual = {}
    }

  }
}

resource "meshstack_api_key" "reader" {
  metadata = {
    owned_by_workspace = data.meshstack_workspace.example.metadata.name
  }

  spec = {
    display_name = "ci-key"
    permissions  = ["MANAGED_BUILDINGBLOCK_SAVE", "MANAGED_BUILDINGBLOCK_LIST", "ADM_BUILDINGBLOCK_DELETE"]
    # expires_at is optional; if omitted the key never expires.
    # Setting an expiry is recommended for security best practices.
    # expires_at = "2025-12-31"
  }
}
