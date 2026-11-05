resource "meshstack_workspace" "test" {
  metadata = {
    name = "test-ws-[[.suffix]]"
  }
  spec = {
    display_name = "Test Workspace"
  }
}

resource "meshstack_workspace_[[.subject_kind]]_binding" "test" {
  metadata = {
    name = "my-workspace-[[.subject_kind]]-binding"
  }
  role_ref = {
    name = "Workspace Member"
  }
  target_ref = {
    name = meshstack_workspace.test.metadata.name
  }
  subject = {
    name = "[[.subject_name]]"
  }
  expiry_date = "2099-12-31"
}

[[ example "data-sources/meshstack_workspace_" .subject_kind "_binding/data-source.tf" ]]
