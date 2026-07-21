# ===========================================================================
# operator/ — everything the PLATFORM OPERATOR does, all through its scoped key.
# Split across two files:
#   bbd.tf          — the building block definition + the operator's own test block
#   managing_bbs.tf — cross-workspace adopt + upgrade of app-team blocks (steps 7-8)
#
#   1-2) author a WORKSPACE_LEVEL terraform BBD (run_transparency=false) and
#        validate it with a test block in its own workspace (secret decrypted).
#   3)   break it (ref_name -> broken).
#   4)   release the broken version as v1.
#   6)   fix it (ref_name -> main) and release v2, which ADDS a DEFAULTED `size`
#        operator input that v1 lacked.
# ===========================================================================

locals {
  # The local-dev-stack seeds this shared runner (multiplexing -> terraform + manual runners).
  runner_uuid = "98520496-627d-43e6-82da-ce499179ff3f"

  # The local tf-block-runner clones the committed bare fixture repo over file://.
  # operator/ is at scratch/bbv3-demo-v2/operator, so ../../../ is the repo root.
  bb_module_repo = "file://${abspath("${path.root}/../../../internal/provider/testdata/tf-building-block")}"

  # bbd_phase -> (terraform ref, draft flag, whether the v2 `size` input is present).
  phase = {
    "draft-good"   = { ref = "main", draft = true, v2 = false }
    "draft-broken" = { ref = "broken", draft = true, v2 = false }
    "v1-released"  = { ref = "broken", draft = false, v2 = false }
    "v2-draft"     = { ref = "main", draft = true, v2 = true }
    "v2-released"  = { ref = "main", draft = false, v2 = true }
  }[var.bbd_phase]

  display_name = "BBv3 Demo v2 — Feature Definition ${var.suffix}"

  # The v2 fix adds a DEFAULTED platform-operator input. Because it is defaulted, the
  # operator's upgrade (steps 7-8) need not supply it — the backend applies `16`.
  operator_inputs = local.phase.v2 ? {
    size = {
      display_name    = "Size"
      type            = "INTEGER"
      assignment_type = "PLATFORM_OPERATOR_MANUAL_INPUT"
      default_value   = jsonencode(16)
    }
  } : {}
}

# ---------------------------------------------------------------------------
# The building block definition (authored + evolved by the operator).
# ---------------------------------------------------------------------------
resource "meshstack_building_block_definition" "feature" {
  metadata = {
    owned_by_workspace = var.platform_workspace
  }
  spec = {
    display_name = local.display_name
    description  = "Workspace-level terraform definition with a sensitive user input (bbv3-demo-v2)."
    target_type  = "WORKSPACE_LEVEL" # targets a workspace directly — no platform/LZ/tenant needed
  }
  version_spec = {
    draft            = local.phase.draft
    run_transparency = false # only platform teams see run logs; the app team cannot (step 5)
    runner_ref       = { kind = "meshBuildingBlockRunner", uuid = local.runner_uuid }

    inputs = merge({
      name = {
        display_name           = "Name"
        type                   = "STRING"
        assignment_type        = "USER_INPUT"
        updateable_by_consumer = true
      }
      environment = {
        display_name           = "Environment"
        type                   = "SINGLE_SELECT"
        assignment_type        = "USER_INPUT"
        updateable_by_consumer = true
        selectable_values      = ["dev", "staging", "prod"]
      }
      # Sensitive USER_INPUT: encrypted to the runner's public key, surfaced only as a hash.
      api_key = {
        display_name           = "API Key"
        description            = "Sensitive API key handed to the terraform implementation."
        type                   = "STRING"
        assignment_type        = "USER_INPUT"
        updateable_by_consumer = true
        sensitive              = {}
      }
      region = {
        display_name    = "Region"
        type            = "STRING"
        assignment_type = "STATIC"
        argument        = jsonencode("eu-west-1")
      }
    }, local.operator_inputs)

    implementation = {
      terraform = {
        terraform_version = "1.9.0"
        repository_url    = local.bb_module_repo
        ref_name          = local.phase.ref # "main" (works) or "broken" (failing precondition)
      }
    }

    outputs = {
      # The module echoes the decrypted api_key here (non-sensitive) — proof of end-to-end decryption.
      api_key_echo = {
        display_name    = "API Key Echo"
        type            = "STRING"
        assignment_type = "NONE"
      }
    }
  }
}

# ---------------------------------------------------------------------------
# Operator's own test block (platform workspace). Validates the BBD before
# release; tracks version_latest so it follows the current (draft or released)
# version. Supplies only user inputs — on v2 the defaulted `size` lands by itself.
# It is ALWAYS present: because the ref carries version_latest.content_hash, editing
# the draft (e.g. the good->broken ref flip) re-runs THIS block in place rather than
# replacing it.
# ---------------------------------------------------------------------------
resource "meshstack_building_block" "operator_test" {
  spec = {
    building_block_definition_version_ref = meshstack_building_block_definition.feature.version_latest
    display_name                          = "operator-test-${var.suffix}"
    target_ref                            = { kind = "meshWorkspace", name = var.platform_workspace }

    inputs = {
      name        = { value = jsonencode("operator-test") }
      environment = { value = jsonencode("dev") }
      api_key = {
        sensitive = {
          secret_value   = "super-secret-api-key"
          secret_version = "1"
        }
      }
    }
  }

  # wait_for_completion / purge_on_delete left at their defaults (true / false). On a good ref the apply
  # waits for SUCCEEDED; on the broken ref it ERRORS and surfaces the run log (operator can read it
  # despite run_transparency=false).
  timeouts = {
    create = "5m"
    update = "5m"
    delete = "5m"
  }
}

output "operator_test_api_key_echo" {
  description = "Decrypted api_key echoed by the module — proves end-to-end sensitive-input decryption."
  value       = meshstack_building_block.operator_test.status.outputs
}

output "operator_test_bb_status" {
  description = "Execution status of the operator test block: SUCCEEDED on a good ref, FAILED on the broken-ref re-run."
  value       = meshstack_building_block.operator_test.status.status
}

output "bbd_state" {
  description = "State of the BBD's latest version: DRAFT while iterating (draft-* / v2-draft phases), RELEASED once published (v1-released / v2-released)."
  value       = meshstack_building_block_definition.feature.version_latest.state
}
