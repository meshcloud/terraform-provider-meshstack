resource "meshstack_platform_type" "example" {
  metadata = {
    # meshStack stores a platform type identifier upper-cased, so the suffix is too — otherwise the
    # value read back would never match the one applied.
    name               = "CUSTOM-PT-${upper(var.suffix)}"
    owned_by_workspace = meshstack_workspace.example.metadata.name
  }

  spec = {
    display_name     = "My Custom Platform ${var.suffix}"
    default_endpoint = "https://platform.example.com"
    icon             = "data:image/svg+xml;base64,PHN2ZyB4bWxucz0iaHR0cDovL3d3dy53My5vcmcvMjAwMC9zdmciLz4="
  }
}
