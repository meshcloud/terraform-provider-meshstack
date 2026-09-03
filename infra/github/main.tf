# This module puts the repository's GitHub configuration under version control. It is deployed by
# hand, like the equivalent modules in meshstack-hub and meshfed-release, and it covers only the
# default-branch ruleset so far.
locals {
  github_repository_name = "terraform-provider-meshstack"

  # Every check that gates a merge, mapped to the app allowed to report it. A check that runs on a
  # pull request but is not listed here cannot block a merge, which is how a pull request stayed
  # mergeable while the acceptance test was still running. Pinning the app means no other
  # integration can report a check under the same name.
  #
  # The first five are jobs of the "Tests" workflow (.github/workflows/test.yml), so their contexts
  # are the job names without the workflow prefix GitHub shows in its UI.
  #
  # The acceptance check is the exception, and it comes from a different app. That suite needs a
  # whole meshStack backend, so meshfed-release runs it and posts the result back here; see
  # .github/workflows/ci-request-integration.yml and the github-ci skill.
  #
  # APPLY ORDER, because this module is deployed by hand. "Go Acceptance Test" no longer exists once
  # the workflow change merges, and "Acceptance Tests (meshStack backend)" is not reported until
  # ci-request-integration.yml sits on the default branch. So apply this immediately before merging
  # that change, and merge right after: in the window between the two, an open pull request based on
  # the old default branch requires a check that nothing reports.
  #
  # A fork pull request gets no acceptance check at all - the dispatcher refuses to spend a dispatch
  # on code it does not trust - so requiring it blocks a fork pull request outright. A fork
  # contribution now needs a maintainer to adopt the branch into this repository.
  required_checks = {
    "Go Build"                             = local.github_actions_app_id
    "Go Lint and Format Check"             = local.github_actions_app_id
    "Shell Lint (CI scripts)"              = local.github_actions_app_id
    "Generate Terraform Provider Docs"     = local.github_actions_app_id
    "Go Test"                              = local.github_actions_app_id
    "Acceptance Tests (meshStack backend)" = local.satellite_app_id
  }

  # GitHub App ids of the two apps that report the checks above. The provider cannot resolve an app
  # slug to an id, so both are pinned by hand. Retrieve either from a commit that carries the check:
  #   gh api /repos/meshcloud/terraform-provider-meshstack/commits/<sha>/check-runs \
  #     --jq '.check_runs[] | {name, app_id: .app.id, app: .app.slug}'
  github_actions_app_id = 15368  # github-actions
  satellite_app_id      = 781479 # meshcloud-gh-actions, which meshfed-release reports with
}

# The ruleset was created by hand in the GitHub UI before this module existed.
import {
  id = "${local.github_repository_name}:11629786"
  to = github_repository_ruleset.protect_default_branch
}

resource "github_repository_ruleset" "protect_default_branch" {
  repository  = local.github_repository_name
  name        = "Protect default branch"
  target      = "branch"
  enforcement = "active"

  # Org admins keep a bypass as an emergency escape hatch, and so does the maintain role
  # (RepositoryRole 2) for the same reason.
  bypass_actors {
    actor_id    = 0 # org admin (role-based; GitHub stores 0)
    actor_type  = "OrganizationAdmin"
    bypass_mode = "always"
  }

  bypass_actors {
    actor_id    = 2
    actor_type  = "RepositoryRole"
    bypass_mode = "always"
  }

  conditions {
    ref_name {
      include = ["~DEFAULT_BRANCH"]
      exclude = []
    }
  }

  rules {
    deletion                = true
    non_fast_forward        = true # force push
    required_linear_history = true

    # The live rule also has require_extra_approval_for_unattributed_changes = true, which the
    # provider cannot express as of 6.13.0 - it is missing from the schema, so it appears in no plan.
    # GitHub kept the setting through the apply that imported this ruleset, so it survives a PUT that
    # omits it. Check it after any apply that changes this rule, and restore it if it ever vanishes:
    #   gh api repos/meshcloud/terraform-provider-meshstack/rulesets/11629786 \
    #     --jq '.rules[] | select(.type == "pull_request")'
    pull_request {
      required_approving_review_count   = 1
      required_review_thread_resolution = true
      allowed_merge_methods             = ["rebase"]
      dismiss_stale_reviews_on_push     = false
      require_code_owner_review         = false
      require_last_push_approval        = false
    }

    required_status_checks {
      # A pull request has to be up to date with the default branch before it can merge, so a
      # check result always describes the code that actually lands.
      strict_required_status_checks_policy = true

      dynamic "required_check" {
        for_each = local.required_checks
        content {
          context        = required_check.key
          integration_id = required_check.value
        }
      }
    }
  }
}
