data "meshstack_projects" "example_all_projects_in_workspace" {
  workspace_identifier = "my-workspace-identifier"
}

data "meshstack_projects" "example_all_projects_in_workspace_with_payment_method" {
  workspace_identifier = "my-workspace-identifier"

  # use empty string "" for listing projects without a payment method, omit for all projects
  payment_method_identifier = "my-payment-method-identifier"
}
