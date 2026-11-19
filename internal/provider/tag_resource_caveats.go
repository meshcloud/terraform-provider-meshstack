package provider

type taggedObject struct {
	name              string
	kind              string
	ownerResource     string
	inlineTags        string
	tagResource       string
	tagsResource      string
	injectedDefaultOn string
}

// tagResourceCaveats documents the dedicated tag resources, which are read-modify-write wrappers
// around a whole meshObject.
func tagResourceCaveats(o taggedObject) string {
	return "!> **Not recommended for general use.** Prefer managing tags inline via `" + o.inlineTags + "` on " +
		"`" + o.ownerResource + "`. Only reach for this resource when the " + o.name + " itself is not managed by your Terraform " +
		"configuration (for example it was created in meshPanel or by another team) and you understand the " +
		"trade-offs below. All of them follow from the same limitation: the meshObject API has no endpoint for individual " +
		"tags, so every create, update and delete here reads the entire `" + o.kind + "` object and writes it back with the " +
		"tags replaced.\n\n" +

		"~> **The entire " + o.name + " is rewritten on every change.** The Terraform plan only shows a tag changing, but each " +
		"apply issues a full " + o.name + " update. Everything meshStack does on a " + o.name + " update — audit-log entries, " +
		"notifications etc. — happens every time, and any " + o.name + " field changed outside Terraform between this " +
		"resource's read and its write is written back with the value it read.\n\n" +

		"~> **Tags you do not manage are written back verbatim.** Preserving a " + o.name + "'s other tags is what lets " +
		"several `" + o.tagResource + "` resources coexist, but the object read back can carry entries nobody " +
		"declared here — notably the defaults meshStack injects for restricted tag definitions on " + o.injectedDefaultOn + " " +
		"— and every write sends all of them back. So this resource writes tag entries " +
		"you never configured, and if your meshStack rejects writing a restricted tag's value, the update fails and no " +
		"tag on that " + o.name + " can be managed with this resource — use inline `" + o.inlineTags + "` on `" + o.ownerResource + "` " +
		"instead. `" + o.tagsResource + "` is unaffected: it sends exactly the tags you configure.\n\n" +

		"~> **Race conditions.** The read-modify-write cycle is not atomic and the API offers no optimistic locking. " +
		"Within a single apply the provider serializes every write to the same " + o.name + " — from this resource, " +
		"`" + o.tagResource + "`, `" + o.tagsResource + "` and `" + o.ownerResource + "` alike — so several " +
		"`" + o.tagResource + "` resources on one " + o.name + " keep each other's tags without `depends_on`. Nothing " +
		"outside that one provider instance is serialized: a concurrent write from another apply, a second provider " +
		"configuration (alias) pointing at the same meshStack, a panel user, or any other automation can silently clobber " +
		"this resource's tags or be clobbered by them. Manage all tags of a " + o.name + " from a single Terraform state " +
		"and never run two applies against the same " + o.name + " in parallel.\n\n" +

		"~> **Cannot manage tags that are mandatory at " + o.name + " creation.** This resource can only set a tag on a " +
		o.name + " that already exists, so it cannot supply tag values that meshStack requires when the " + o.name + " is " +
		"created — the `" + o.ownerResource + "` create fails before this resource ever runs. Set mandatory tags inline via " +
		"`" + o.inlineTags + "` on `" + o.ownerResource + "`.\n\n" +

		"~> **Subject to change.** This resource is provisional. Once meshStack's meshObject API supports " + o.name + " tags " +
		"as individual meshObjects, it will be reworked in terms of that API."
}
