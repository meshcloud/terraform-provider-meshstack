resource "meshstack_platform_type" "custom" {
  metadata = {
    name               = "MY-CUSTOM-PLATFORM-TYPE-${random_string.platform_type_suffix.result}"
    owned_by_workspace = "my-workspace"
  }

  spec = {
    display_name     = "My Custom Platform ${random_string.platform_type_suffix.result}"
    default_endpoint = "https://platform.example.com"
    icon             = "data:image/svg+xml;base64,PHN2ZyB4bWxucz0iaHR0cDovL3d3dy53My5vcmcvMjAwMC9zdmciLz4="
  }
}

resource "random_string" "platform_type_suffix" {
  length  = 8
  upper   = true
  lower   = false
  special = false
  numeric = false
}
