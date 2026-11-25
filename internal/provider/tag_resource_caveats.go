package provider

func tagResourceCaveats(o tagTargetInfo, tagResource string) string {
	name := o.name
	ownerResource := "meshstack_" + name
	tagsResource := ownerResource + "_tags"
	return "!> **Not recommended for general use.** Prefer managing tags inline via `" + o.inlineTags + "` on " +
		"`" + ownerResource + "`. Only reach for this resource when the " + name + " itself is not managed by your Terraform " +
		"configuration (for example it was created in meshPanel or by another team) and you understand the " +
		"trade-offs below. All of them follow from the same limitation: the meshObject API has no endpoint for individual " +
		"tags, so every create, update and delete here reads the entire `" + o.kind + "` object and writes it back with the " +
		"tags replaced.\n\n" +

		"~> **The entire " + name + " is rewritten on every change.** The Terraform plan only shows a tag changing, but each " +
		"apply issues a full " + name + " update. Everything meshStack does on a " + name + " update — audit-log entries, " +
		"notifications etc. — happens every time, and any " + name + " field changed outside Terraform between this " +
		"resource's read and its write is written back with the value it read.\n\n" +

		"~> **Tags you do not manage are written back verbatim.** Preserving a " + name + "'s other tags is what lets " +
		"several `" + tagResource + "` resources coexist, but the object read back can carry entries nobody " +
		"declared here — notably the defaults meshStack injects for restricted tag definitions on " + o.injectedDefaultOn + " " +
		"— and every write sends all of them back. So this resource writes tag entries you never configured. " +
		"`" + tagsResource + "` sends exactly the tags you configure.\n\n" +

		"~> **Restricted tags need an API key with admin rights.** Only an API key with admin rights can change a tag " +
		"whose definition is restricted. For any other API key, meshStack keeps the stored values of the restricted tags " +
		"and ignores changes to them without an error; for an admin API key, it removes every tag the write leaves out. " +
		"This resource compares the tags meshStack stored with the tags it manages after each write and fails when " +
		"meshStack ignored the change.\n\n" +

		"~> **Race conditions.** The read-modify-write cycle is not atomic and the API offers no optimistic locking. " +
		"Within a single apply the provider serializes every write to the same " + name + " — from this resource, " +
		"the other " + name + " tag resources and `" + ownerResource + "` alike — so several " +
		"`" + tagResource + "` resources on one " + name + " keep each other's tags without `depends_on`. Nothing " +
		"outside that one provider instance is serialized: a concurrent write from another apply, a second provider " +
		"configuration (alias) pointing at the same meshStack, a panel user, or any other automation can silently clobber " +
		"this resource's tags or be clobbered by them. Manage all tags of a " + name + " from a single Terraform state " +
		"and never run two applies against the same " + name + " in parallel.\n\n" +

		"~> **Cannot manage tags that are mandatory at " + name + " creation.** This resource can only set a tag on a " +
		name + " that already exists, so it cannot supply tag values that meshStack requires when the " + name + " is " +
		"created — the `" + ownerResource + "` create fails before this resource ever runs. Set mandatory tags inline via " +
		"`" + o.inlineTags + "` on `" + ownerResource + "`.\n\n" +

		"~> **Subject to change.** This resource is provisional. Once meshStack's meshObject API supports " + name + " tags " +
		"as individual meshObjects, it will be reworked in terms of that API."
}
