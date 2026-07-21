# ===========================================================================
# 00_bootstrap (admin) — the only things the admin creates:
#   * two workspaces (platform, appteam)
#   * two least-privilege API keys (operator, appteam)
#
# No platform/landing-zone/tenant: the demo BBD is WORKSPACE_LEVEL and targets a
# workspace directly. The operator authors the BBD itself in operator/ with its
# own key; the app team consumes it in appteam/ with its own key. Outputs here
# are sourced into TF_VAR_* for those two states (see README).
# ===========================================================================

# Short random suffix so the demo can be re-run after teardown without name
# collisions. Carried to the other states via the `suffix` output / TF_VAR_suffix
# (the BBD display name embeds it, and the app team searches by that name).
resource "random_string" "suffix" {
  length  = 4
  lower   = true
  upper   = false
  numeric = false
  special = false
}

locals {
  suffix = random_string.suffix.result
}

resource "meshstack_workspace" "platform" {
  metadata = {
    name = "bbv3-demo-v2-platform-${local.suffix}"
    tags = {}
  }
  spec = {
    display_name = "BBv3 Demo v2 — Platform ${local.suffix}"
  }
}

resource "meshstack_workspace" "appteam" {
  metadata = {
    name = "bbv3-demo-v2-appteam-${local.suffix}"
    tags = {}
  }
  spec = {
    display_name = "BBv3 Demo v2 — App Team ${local.suffix}"
  }
}

# Platform-operator key, owned by the PLATFORM workspace (which owns the BBD).
#   * BUILDINGBLOCKDEFINITION_{SAVE,LIST,DELETE} -> author/release/break/fix the BBD in its own ws
#   * ADM_BUILDINGBLOCKDEFINITION_SAVE + ADM_REVIEW_PUBLICATION -> release versions WITHOUT admin
#                                                  approval. The release here flips an existing draft
#                                                  true->false, which hits the PUT updateVersion path;
#                                                  that path bypasses requiresAdmApprovalForRelease only
#                                                  when the caller has BOTH authorities (hasAllOfAuthorities).
#                                                  Without them the BBD parks in a pending-approval state.
#   * BUILDINGBLOCK_{SAVE,LIST,DELETE}           -> its own test block (operator_test)
#   * MANAGED_BUILDINGBLOCK_{SAVE,LIST}          -> reach blocks made from its BBD in OTHER workspaces
#   * MANAGED_BUILDINGBLOCKRUN_LIST              -> view runs/logs of blocks made from its BBD as the
#                                                  definition owner, even when run_transparency=false
#                                                  (this is what lets the operator see the failed run's
#                                                  log on the broken ref — canViewRuns() owner path)
# Deliberately NO delete authority over other workspaces' blocks: the operator
# adopts + upgrades the app team's block but never deletes it.
resource "meshstack_api_key" "operator" {
  metadata = {
    owned_by_workspace = meshstack_workspace.platform.metadata.name
  }
  spec = {
    display_name = "bbv3-demo-v2-operator-key-${local.suffix}"
    permissions = [
      "BUILDINGBLOCKDEFINITION_SAVE", "BUILDINGBLOCKDEFINITION_LIST", "BUILDINGBLOCKDEFINITION_DELETE",
      "ADM_BUILDINGBLOCKDEFINITION_SAVE", "ADM_REVIEW_PUBLICATION",
      "BUILDINGBLOCK_SAVE", "BUILDINGBLOCK_LIST", "BUILDINGBLOCK_DELETE",
      "MANAGED_BUILDINGBLOCK_SAVE", "MANAGED_BUILDINGBLOCK_LIST", "MANAGED_BUILDINGBLOCKRUN_LIST",
    ]
  }
}

# App-team key, owned by the APP TEAM workspace.
#   * BUILDINGBLOCK_{SAVE,LIST,DELETE} -> create/update/delete its own block
#   * BUILDINGBLOCKDEFINITION_LIST     -> find the operator's BBD by name (cross-workspace LIST;
#                                         version content_hash comes back null, which is fine)
# Intentionally cannot change the BBD version (no operator/admin authority).
resource "meshstack_api_key" "appteam" {
  metadata = {
    owned_by_workspace = meshstack_workspace.appteam.metadata.name
  }
  spec = {
    display_name = "bbv3-demo-v2-appteam-key-${local.suffix}"
    permissions  = ["BUILDINGBLOCK_SAVE", "BUILDINGBLOCK_LIST", "BUILDINGBLOCK_DELETE", "BUILDINGBLOCKDEFINITION_LIST"]
  }
}

# ---------------------------------------------------------------------------
# Outputs — source these into TF_VAR_* before applying operator/ and appteam/.
# ---------------------------------------------------------------------------
output "platform_workspace" {
  value = meshstack_workspace.platform.metadata.name
}

output "appteam_workspace" {
  value = meshstack_workspace.appteam.metadata.name
}

output "suffix" {
  value = local.suffix
}

output "operator_client_id" {
  value     = meshstack_api_key.operator.status.client_id
  sensitive = true
}

output "operator_client_secret" {
  value     = meshstack_api_key.operator.status.client_secret
  sensitive = true
}

output "appteam_client_id" {
  value     = meshstack_api_key.appteam.status.client_id
  sensitive = true
}

output "appteam_client_secret" {
  value     = meshstack_api_key.appteam.status.client_secret
  sensitive = true
}
