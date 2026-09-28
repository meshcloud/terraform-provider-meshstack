data "meshstack_service_instance" "example" {
  metadata = {
    # No Terraform resource creates a service instance, so this id is one the mock store is seeded
    # with rather than one another block produces.
    instance_id = "test-instance-id"
  }
}
