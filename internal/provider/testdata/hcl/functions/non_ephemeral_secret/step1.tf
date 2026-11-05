output "secret" {
  value = provider::meshstack::non_ephemeral_secret("[[.value]]")
}
