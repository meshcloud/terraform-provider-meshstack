terraform {
  required_version = ">= 1.0"

  required_providers {
    github = {
      source  = "integrations/github"
      version = "~> 6.13"
    }
  }

  backend "gcs" {
    bucket = "meshcloud-tf-states"
    prefix = "terraform-provider-meshstack/infra/github"
  }
}

provider "github" {
  owner = "meshcloud"
}
