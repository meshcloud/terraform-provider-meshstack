# ===========================================================================
# Steps 7-8 — cross-workspace adopt + upgrade (enabled via TF_VAR_manage_appteam=true).
#
# List every block created from THIS definition across workspaces (operator scope,
# needs MANAGED_BUILDINGBLOCK_LIST), EXCLUDE the operator's own test block (platform workspace, already
# managed in bbd.tf), import each remaining block into operator state, and bump it to the released v2.
#
# PLAN-TIME CONSTRAINT: the `import` block below uses `for_each = local.managed`, and an import
# for_each must be resolvable at PLAN time. local.managed comes from this data source, which filters
# on `feature` (defined in bbd.tf) — so OpenTofu only reads it during plan when `feature` has NO
# pending change. Release v2 (step 6b) in its own apply FIRST so `feature` is settled; otherwise the
# data source defers to apply, local.managed is unknown at plan, and the import for_each errors. If
# that happens, settle the BBD first: `tofu apply -target=meshstack_building_block_definition.feature`
# then `tofu apply`.
# ===========================================================================
data "meshstack_building_blocks" "managed" {
  count = var.manage_appteam ? 1 : 0

  managed_by_definition_uuid = meshstack_building_block_definition.feature.metadata.uuid
}

locals {
  managed = var.manage_appteam ? {
    # Exclude the operator's OWN test block (in the platform workspace) — it is managed directly in
    # bbd.tf, so adopting it here would be a self-import. Only adopt blocks in OTHER (app-team)
    # workspaces. In this demo that leaves exactly one block: the app team's.
    for bb in data.meshstack_building_blocks.managed[0].building_blocks : bb.metadata.uuid => bb
    if bb.spec.target_ref.name != var.platform_workspace
  } : {}
}

import {
  for_each = local.managed
  to       = meshstack_building_block.managed[each.key]
  id       = each.value.metadata.uuid
}

resource "meshstack_building_block" "managed" {
  for_each = local.managed

  spec = {
    # Pure version bump to the released v2 (the fix). The new `size` input is
    # defaulted, so the operator declares NO inputs — `size = 16` is applied by
    # the backend, and the app team's user inputs (incl. api_key) are preserved.
    building_block_definition_version_ref = meshstack_building_block_definition.feature.version_latest_release
    display_name                          = each.value.spec.display_name
    target_ref                            = { kind = "meshWorkspace", name = each.value.spec.target_ref.name }
    inputs                                = {}
  }

  timeouts = {
    create = "5m"
    update = "5m"
    delete = "5m"
  }

  # The operator adopts these blocks only to UPGRADE them — it must never delete them,
  # they belong to the app team. `destroy = false` (an OpenTofu lifecycle customization;
  # this demo runs OpenTofu) makes OpenTofu FORGET each instance from operator state
  # instead of issuing a delete API call — both on `tofu destroy` and when an instance
  # leaves config (e.g. setting TF_VAR_manage_appteam=false empties the for_each). No
  # `removed` block or manual `tofu state rm` is needed for teardown; see the README.
  lifecycle {
    destroy = false
  }
}

# Which app-team blocks the operator currently adopts + upgrades, as "<uuid> — <display_name>".
# `null` (so OpenTofu omits it from `tofu output`) until TF_VAR_manage_appteam=true; once managing,
# an empty list means the data source found no blocks made from this definition yet.
output "managed_building_blocks" {
  description = "App-team building blocks currently managed by the operator (\"<uuid> — <display_name>\"); null until TF_VAR_manage_appteam=true."
  value = var.manage_appteam ? [
    for bb in meshstack_building_block.managed : "${bb.metadata.uuid} — ${bb.spec.display_name}"
  ] : null
}
