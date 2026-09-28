# The sensitive api_key a building block supplies and the version it declares it under. The backend
# returns a hash and never the value, so a bumped version is the only thing that makes a rotation
# visible to the provider.

variable "secret_value" {
  type      = string
  sensitive = true
  default   = "super-secret-api-key"
}

variable "secret_version" {
  type    = string
  default = "1"
}
