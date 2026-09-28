data "meshstack_platform" "example" {
  metadata = {
    uuid = meshstack_platform.example_azure.metadata.uuid
  }
}
