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

resource "meshstack_payment_method" "test" {
  metadata = {
    name               = "test-pm-[[.suffix]]"
    owned_by_workspace = meshstack_workspace.test.metadata.name
  }
  spec = {
    display_name    = "Test Payment Method"
    expiration_date = "2099-12-31"
  }
}

data "meshstack_payment_method" "example" {
  metadata = {
    name               = meshstack_payment_method.test.metadata.name
    owned_by_workspace = meshstack_payment_method.test.metadata.owned_by_workspace
  }
}

resource "meshstack_building_block_definition" "example" {
  metadata = {
    owned_by_workspace = data.meshstack_workspace.example.metadata.name
  }

  spec = {
    display_name = "Test BB v3 Definition"
    description  = "A building block definition for BB v3 resource tests"
  }

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
        description            = "Size of the resource"
        type                   = "INTEGER"
        assignment_type        = "USER_INPUT"
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
      payment_method = {
        display_name           = "Payment Method"
        type                   = "CODE"
        assignment_type        = "PAYMENT_METHOD"
        updateable_by_consumer = true
      }
      region = {
        display_name    = "Region"
        type            = "STRING"
        assignment_type = "STATIC"
        argument        = jsonencode("eu-west-1")
      }
      enable_monitoring = {
        display_name    = "Enable Monitoring"
        type            = "BOOLEAN"
        assignment_type = "STATIC"
        argument        = jsonencode(true)
      }
    }

    implementation = {
      manual = {}
    }

  }
}

data "meshstack_building_block_definition" "example" {
  metadata = {
    uuid = meshstack_building_block_definition.example.metadata.uuid
  }
}

resource "meshstack_building_block" "example_workspace" {
  spec = {
    # Alternatively, use version_latest_release to target only released versions
    building_block_definition_version_ref = data.meshstack_building_block_definition.example.version_latest

    display_name = "my-workspace-building-block-renamed"
    target_ref   = data.meshstack_workspace.example.ref

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
      payment_method = {
        value = jsonencode(data.meshstack_payment_method.example.ref)
      }
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
}
