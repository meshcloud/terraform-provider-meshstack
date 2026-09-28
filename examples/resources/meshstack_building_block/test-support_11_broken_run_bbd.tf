# The sensitive definition pointed at the bare repo's `broken` branch, whose module fails `tofu
# apply` on a deliberately-false precondition. run_transparency is the thing under test: it decides
# whether a workspace-scoped key may trigger the repair run at all.
resource "meshstack_building_block_definition" "sensitive_user_input" {
  metadata = {
    owned_by_workspace = meshstack_workspace.example.metadata.name
  }

  spec = {
    display_name     = "BB v3 Broken Run Test Definition"
    description      = "Definition whose terraform implementation always fails, for the run-transparency scenario"
    run_transparency = var.run_transparency
  }

  version_spec = {
    draft = false

    # A plain DELETE by the consumer's workspace key soft-deletes the block without a destroy run. The
    # default mode would run the broken module again at teardown, re-fail the precondition and leave
    # the block stuck in FAILED — a nondeterministic post-destroy failure under the parallel suite.
    deletion_mode = "PURGE"

    inputs = {
      api_key = {
        display_name    = "API Key"
        type            = "STRING"
        assignment_type = "USER_INPUT"
        sensitive       = {}
      }
      script = {
        display_name    = "Startup Script"
        type            = "CODE"
        assignment_type = "USER_INPUT"
        sensitive       = {}
      }
    }

    implementation = {
      terraform = {
        terraform_version = "1.9.0"
        repository_url    = var.terraform_repository_url
        ref_name          = "broken"
      }
    }

    outputs = {}
  }
}
