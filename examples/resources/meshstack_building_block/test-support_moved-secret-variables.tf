# The api_key a moved block re-declares. The defaults are the placeholder that must NOT reach the
# backend; a rotation dials a real new value under a bumped version, which is what re-applies the
# secret and changes its hash.

variable "moved_secret_value" {
  type      = string
  sensitive = true
  default   = "placeholder-not-the-real-api-key"
}

# Null means no version at all, which is what preserves the stored secret.
variable "moved_secret_version" {
  type    = string
  default = null
}
