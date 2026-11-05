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
    name = "test-wb-[[.suffix]]"
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
}
