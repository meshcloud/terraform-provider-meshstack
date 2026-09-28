# The git smart-HTTP URL of the committed bare repo the tf-block-runner clones, so a terraform
# building block actually runs OpenTofu offline instead of leaving a stuck run behind. The test
# supplies it because the loopback port is only known once the test's git server is listening; in
# mock mode nothing clones, so the value is unused.

variable "terraform_repository_url" {
  type = string
}
