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

# An example for a manual implementation, showing a sparse output override.
resource "meshstack_building_block_definition" "example_03_manual" {
  metadata = {
    owned_by_workspace = data.meshstack_workspace.example.metadata.name
  }

  spec = {
    display_name = "Example Building Block"
    description  = "An example building block definition"
  }

  version_spec = {
    draft = true

    inputs = {
      approval = { display_name = "Approval", type = "BOOLEAN", assignment_type = "PLATFORM_OPERATOR_MANUAL_INPUT" }
      region   = { display_name = "Region", type = "SINGLE_SELECT", assignment_type = "USER_INPUT", selectable_values = ["eu", "us"] }
    }

    implementation = {
      manual = {}
    }

    # Manual building blocks are special: the backend derives one output per input, and each output's
    # `type` is always taken from its input (it must NOT be set here). `version_spec.outputs` is therefore
    # a SPARSE OVERRIDE — declare only the outputs you want to customize and leave the rest to be derived.
    # For a declared output you may set `assignment_type` (to mark how the value is used) and, optionally,
    # `display_name` and `display_order`. Omit the attribute entirely (or set `{}`) to customize nothing.
    outputs = {
      region = { assignment_type = "SUMMARY" }
    }
  }
}
