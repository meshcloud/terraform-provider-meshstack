resource "meshstack_workspace" "test" {
  metadata = {
    name = "test-ws-[[.suffix]]"
  }
  spec = {
    display_name = "Test Workspace"
  }
}

resource "meshstack_project" "test" {
  metadata = {
    name               = "test-proj-[[.suffix]]"
    owned_by_workspace = meshstack_workspace.test.metadata.name
  }
  spec = {
    display_name = "Test Project"
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

resource "meshstack_project_[[.subject_kind]]_binding" "test" {
  metadata = {
    name = "my-project-[[.subject_kind]]-binding"
  }
  role_ref = {
    name = "Project Reader"
  }
  target_ref = {
    owned_by_workspace = meshstack_project.test.metadata.owned_by_workspace
    name               = meshstack_project.test.metadata.name
  }
  subject = {
    name = meshstack_workspace_[[.subject_kind]]_binding.test.subject.name
  }
  expiry_date = "2099-12-31"
}

data "meshstack_project_[[.subject_kind]]_binding" "example" {
  target_ref = meshstack_project_[[.subject_kind]]_binding.test.target_ref
  subject    = meshstack_project_[[.subject_kind]]_binding.test.subject
  role_ref   = { name = "Project Reader" }
}
