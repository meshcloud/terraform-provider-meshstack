---
name: user-facing-wording
description: Find and fix wording inconsistencies in this provider's user-facing text (diagnostics, schema descriptions, example comments, changelog prose, docs) — against meshStack itself and against the provider's own other surfaces. Use when writing or changing an AddError/AddAttributeError message, a MarkdownDescription, a client error string, an example .tf comment, a CHANGELOG entry, or when asked to review or align wording.
---

# Wording consistency

This provider is a second voice for meshStack's concepts. The same rule is stated in the backend's
API errors, in the meshObject API field docs, in the meshPanel, on docs.meshcloud.io — and again
here, in a diagnostic or a Registry doc. meshStack is the source of truth: where this repo's wording
disagrees with meshStack, or with itself, that is the defect to fix.

## Determine the scope

Before searching, settle what part of the codebase to check:

- If the current conversation already implies a scope — a diff just written, a PR under discussion,
  a feature being built — use that.
- Otherwise, ask the user what to check: a section of code, a PR, a recent change, or the whole
  repository.

Do not default to a full-repo sweep without confirming that is what's wanted; it's expensive and
usually not the intent.

## What counts as user-facing text

Anything a user or Registry reader ends up reading: diagnostic summaries and details, schema
descriptions, example config comments, client error strings, changelog entries, and any other
generated or published documentation. Treat a change to any of these as seriously as a change to
docs — several of them are published verbatim.

## Finding meshStack's wording

meshStack's backend lives in `../meshfed-release`, relative to this repository's root, by the usual
sibling-checkout convention. If the scope in question is a feature still under development there,
its code may not be on `../meshfed-release`'s checked-out branch — it could be sitting in another
git worktree instead. Locate that worktree, or ask the user where to find it, rather than assuming
the sibling checkout is up to date.

## Procedure

1. Within the determined scope, collect the user-facing strings and the domain terms they use.
2. For each term, check how meshStack itself describes the same concept elsewhere — same verb, same
   singular/plural treatment, same casing, same value spelling (e.g. what a user actually writes in
   HCL, not an internal enum name).
3. Check the same term across this provider's own surfaces, not just against meshStack — two
   sentences in this repo disagreeing with each other is the same defect as disagreeing with
   upstream.
4. Fix any wording that diverges, and re-verify: rerun any generation step that produces published
   docs from the changed source, and re-check tests that assert on the exact string.

Verbatim matching is not the goal. Different grammar, phrasing, or length is fine — the surfaces
differ in format and audience — as long as the rephrase doesn't violate the source-of-truth
principle or describe the same concept in different words. Stick to the established term for a
concept rather than introducing a synonym.
