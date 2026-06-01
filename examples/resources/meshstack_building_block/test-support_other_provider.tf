variable "apikey_client_id" {
  type     = string
  nullable = false
}

variable "apikey_client_secret" {
  type      = string
  nullable  = false
  sensitive = true
}

provider "meshstack-other" {
  apikey    = var.apikey_client_id
  apisecret = var.apikey_client_secret
}
