data "meshstack_user" "by_name" {
  metadata = {
    name = "[[.seeded_username]]"
  }
}

data "meshstack_user" "by_email" {
  spec = {
    email = data.meshstack_user.by_name.spec.email
  }
}
