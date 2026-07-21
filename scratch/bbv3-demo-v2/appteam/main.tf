# ===========================================================================
# appteam/ — the APP TEAM consumes the operator's BBD from ANOTHER workspace,
# using only its least-privilege key.
#
#   5) create a block from the released v1 (broken) -> run FAILS. The app team
#      sees only an opaque latest_run_uuid; the run's logs stay gated
#      (run_transparency=false + unprivileged key), so status is FAILED but the
#      failure detail is not visible to the app team.
#   9) after the operator upgrades the block to v2 externally (operator/ steps
#      7-8), a plan shows the version drifted; repin to v2 to reconcile.
# ===========================================================================

# The plural definitions data source has no name filter, so list and filter by
# display name in HCL. Cross-workspace with the app team's scoped key: versions
# come back with uuid/number/state, but content_hash is null (expected).
data "meshstack_building_block_definitions" "all" {}

locals {
  feature = one([
    for d in data.meshstack_building_block_definitions.all.building_block_definitions
    : d if d.spec.display_name == "BBv3 Demo v2 — Feature Definition ${var.suffix}"
  ])

  # versions[] is sorted ascending: [0] = v1, [1] = v2.
  pinned = var.pin == "v1" ? local.feature.versions[0] : local.feature.versions[1]
}

resource "meshstack_building_block" "app_block" {
  spec = {
    building_block_definition_version_ref = { uuid = local.pinned.uuid }
    display_name                          = "app-team-feature-${var.suffix}"
    target_ref                            = { kind = "meshWorkspace", name = var.appteam_workspace }

    inputs = {
      name        = { value = jsonencode("app-team-app") }
      environment = { value = jsonencode("dev") }
      api_key = {
        sensitive = {
          secret_value   = "super-secret-api-key"
          secret_version = "2"
        }
      }
    }
  }

  # wait_for_completion = false on purpose. On v1 the run FAILS (broken ref); if we waited, the failed
  # run would error the CREATE and Terraform would mark the block TAINTED -> step 9 would destroy+recreate
  # it instead of reconciling the operator's external v2 upgrade in place. With false the apply succeeds
  # and the block sits cleanly in state as FAILED (its latest_run_uuid is an opaque id; the run logs
  # stay gated by run_transparency=false). Inspect with
  # `tofu refresh && tofu output app_block_status`. (purge_on_delete left at its default false.)
  wait_for_completion = false
}

output "app_block_status" {
  description = "Expect FAILED on v1 (broken); SUCCEEDED once the operator upgrades to v2."
  value       = meshstack_building_block.app_block.status.status
}

output "app_block_version_uuid" {
  description = "The version the block currently runs; moves from v1 to v2 after the operator upgrade."
  value       = meshstack_building_block.app_block.spec.building_block_definition_version_ref.uuid
}

output "app_block_latest_run_uuid" {
  description = "Opaque run id exposed to the app team; the run's logs stay gated by run_transparency=false."
  # Populated with an opaque uuid once a run exists, but null in the brief window right after create
  # (block PENDING, no run yet). OpenTofu DROPS null-valued outputs from state, so a bare
  # `status.latest_run_uuid` would make `tofu output app_block_latest_run_uuid` error with "output not
  # found" during that window. Coalesce to a sentinel so the output is always present.
  value = coalesce(meshstack_building_block.app_block.status.latest_run_uuid, "<no run yet>")
}
