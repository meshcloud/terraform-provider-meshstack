resource "terraform_data" "broken" {
  lifecycle {
    # OpenTofu rejects a constant condition at validate, so the run would fail at init.
    precondition {
      condition     = terraform_data.noop.output == "force-bbv3-broken-precondition-failure"
      error_message = "bbv3-demo: intentionally broken BBD version (terraform ref_name=broken)."
    }
  }
}
