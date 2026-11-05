locals {
  some_sensitive_value = sensitive("mock-pat-token-12345")
}

output "secret" {
  value = provider::meshstack::non_ephemeral_secret(nonsensitive(local.some_sensitive_value))
}
