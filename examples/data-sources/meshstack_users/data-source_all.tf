data "meshstack_users" "all" {}

locals {
  wanted_members    = ["jane.doe@example.com", "john.doe@example.com"]
  known_usernames   = toset(data.meshstack_users.all.users[*].metadata.name)
  unknown_usernames = setsubtract(local.wanted_members, local.known_usernames)
}

check "members_are_known_users" {
  assert {
    condition     = length(local.unknown_usernames) == 0
    error_message = "meshStack knows no user with the username ${join(", ", local.unknown_usernames)}."
  }
}
