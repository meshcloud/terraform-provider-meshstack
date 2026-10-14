# No-op module: no providers and no backend, so the runner only downloads OpenTofu.

terraform {
  required_version = ">= 1.5.0"
}

# The sensitive inputs of the BBDs in examples/resources/meshstack_building_block/.
variable "api_key" {
  type      = string
  sensitive = true
  default   = null
}

variable "script" {
  type      = string
  sensitive = true
  default   = null
}

variable "static_secret" {
  type      = string
  sensitive = true
  default   = null
}

resource "terraform_data" "noop" {
  input = "bbv3-acc-noop"
}

# Proves that the runner decrypted the sensitive input.
output "api_key_echo" {
  description = "Non-sensitive echo of the decrypted api_key, used to verify sensitive-input decryption."
  value       = nonsensitive(var.api_key)
}
