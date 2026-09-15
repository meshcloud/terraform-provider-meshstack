data "meshstack_building_block_runner" "example" {
  # Omit metadata entirely to read the shared runner meshStack hosts, which is also the runner a
  # building block definition uses when its version_spec.runner_ref is omitted.
  metadata = {
    uuid = meshstack_building_block_runner.example.metadata.uuid
  }
}

# Trust the runner in a cloud backplane: issuer and audience describe the runner and are known before
# any building block definition exists, while the subject is resolved per definition and read there.
locals {
  oidc_issuer  = data.meshstack_building_block_runner.example.spec.workload_identity_federation.issuer
  aws_audience = data.meshstack_building_block_runner.example.spec.workload_identity_federation.aws.audience
  aws_subject  = meshstack_building_block_definition.example.status.workload_identity_federation.subject
}
