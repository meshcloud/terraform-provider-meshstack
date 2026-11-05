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
      tier = {
        display_name           = "Tier"
        description            = "A defaulted platform operator input"
        type                   = "STRING"
        assignment_type        = "PLATFORM_OPERATOR_MANUAL_INPUT"
        default_value          = jsonencode("bronze")
        updateable_by_consumer = true
      }
    }

    implementation = {
      manual = {}
    }

  }
}

variable "apikey_client_id" {
  type     = string
  nullable = false
}

variable "apikey_client_secret" {
  type      = string
  nullable  = false
  sensitive = true
}

provider "meshstack-other" {
  apikey    = var.apikey_client_id
  apisecret = var.apikey_client_secret
}

resource "meshstack_building_block" "example_workspace" {
  spec = {
    # Alternatively, use version_latest_release to target only released versions
    building_block_definition_version_ref = { uuid = meshstack_building_block_definition.example.version_latest_release.uuid }

    display_name = "my-workspace-building-block"
    target_ref   = meshstack_workspace.other.ref

    inputs = {
      name        = { value = jsonencode("updated-name") }
      size        = { value = jsonencode(16) }
      environment = { value = jsonencode("dev") }
    }

    # Building blocks can depend on each other: a parent's outputs feed this block's inputs.
    # Reference a parent by its computed `ref`.
    # parent_building_block_refs = [meshstack_building_block.parent.ref]
  }

  # Purging is a last resort option for stuck deletions. Prefer regular delete behavior.
  # purge_on_delete = true

  timeouts = {
    create = "5m" # default 30m
    update = "5m" # default 30m
    delete = "5m" # default 30m
  }

  # The provider only warns about a run that does not succeed; this fails the apply without tainting the block.
  lifecycle {
    postcondition {
      condition     = self.status.succeeded
      error_message = "Building block ${self.metadata.uuid} did not succeed (health status ${self.status.health_status}, action ${self.status.action}). See its run in meshPanel."
    }
  }
  depends_on = [meshstack_api_key.reader]
}
