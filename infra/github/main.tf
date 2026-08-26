# This module puts the repository's GitHub configuration under version control. It is deployed by
# hand, like the equivalent modules in meshstack-hub and meshfed-release, and it covers only the
# default-branch ruleset so far.
locals {
  github_repository_name = "terraform-provider-meshstack"

  # Every job of the "Tests" workflow (.github/workflows/test.yml). A check that runs on a pull
  # request but is not listed here cannot block a merge, which is how a pull request stayed
  # mergeable while the acceptance test was still running. The contexts are the job names, without
  # the workflow prefix GitHub shows in its UI, and they are pinned to the GitHub Actions app so no
  # other integration can report them.
  #
  # The acceptance job is skipped on a pull request from a fork, because forks have neither the
  # self-hosted runners nor the registry variable it needs. GitHub counts a job skipped by an `if`
  # condition as successful, so requiring it does not block such a pull request - it only means the
  # gate passes there without the tests having run.
  required_checks = [
    "Go Build",
    "Go Lint and Format Check",
    "Shell Lint (CI scripts)",
    "Generate Terraform Provider Docs",
    "Go Test",
    "Go Acceptance Test",
  ]

  # GitHub App id of the GitHub Actions app, which reports every check above. The provider cannot
  # resolve an app slug to an id, so it is pinned by hand. Retrieve via:
  #   gh api repos/meshcloud/terraform-provider-meshstack/rulesets/11629786 \
  #     --jq '.rules[] | select(.type == "required_status_checks")'
  github_actions_app_id = 15368
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
        for_each = toset(local.required_checks)
        content {
          context        = required_check.value
          integration_id = local.github_actions_app_id
        }
      }
    }
  }
}
