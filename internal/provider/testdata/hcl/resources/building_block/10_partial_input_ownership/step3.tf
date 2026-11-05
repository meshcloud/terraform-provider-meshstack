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

resource "meshstack_building_block" "example_workspace" {
  spec = {
    # Alternatively, use version_latest_release to target only released versions
    building_block_definition_version_ref = meshstack_building_block_definition.example.version_latest

    display_name = "my-workspace-building-block"
    target_ref   = data.meshstack_workspace.example.ref

    inputs = { size = { value = jsonencode(16) } }

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
}
