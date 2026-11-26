resource "meshstack_workspace" "test" {
  metadata = {
    name = "test-ws-[[.suffix]]"
  }
  spec = {
    display_name = "Test Workspace"
  }
}

resource "meshstack_workspace_user_group" "test" {
  metadata = {
    name               = "group-[[.suffix]]"
    owned_by_workspace = meshstack_workspace.test.metadata.name
  }
  spec = {
    display_name = "Group [[.suffix]]"
    egid         = "egid-[[.suffix]]"
  }
}

data "meshstack_workspace_user_groups" "by_egid" {
  egid = meshstack_workspace_user_group.test.spec.egid
}

data "meshstack_workspace_user_groups" "by_display_name_in_other_case" {
  owned_by_workspace = meshstack_workspace_user_group.test.metadata.owned_by_workspace
  display_name       = upper(meshstack_workspace_user_group.test.spec.display_name)
}

[[ example "data-sources/meshstack_workspace_user_groups/data-source.tf" ]]
