data "meshstack_platforms" "published" {
  # depends_on, because no attribute of the listing references the platform: without it the query
  # could run before the platform exists and list nothing.
  depends_on = [meshstack_platform.example_custom]

  owned_by_workspace = meshstack_workspace.example.metadata.name
  publication_state  = "PUBLISHED"
}
