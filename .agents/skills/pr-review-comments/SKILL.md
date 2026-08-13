---
name: pr-review-comments
description: Fetch, analyze, and interactively address GitHub PR review comments. User triggers it manually.
---

## Workflow

- [ ] Check for availability of GitHub MCP Server
- [ ] Disable the GH pager (see [[gh-cli-pager]])
- [ ] Fetch repository info and PR number
- [ ] Fetch all inline and top-level review comments, plus review threads (with `isResolved` + thread IDs)
- [ ] Analyze threads to filter out resolved and purely informational ones
- [ ] Address each actionable comment interactively (or automatically in YOLO mode)
- [ ] Reply to each addressed comment and mark its thread resolved
- [ ] Summarize changes made and suggest next steps

## Step 0: Check existence of MCP GitHub Server

**CRITICAL**: If there is an MCP server available use it to access GitHub and its PR comments. Ignore step 1 to 3 and just use the MCP server.

## Step 1: Configure Pager

**CRITICAL**: Before any `gh` command, disable the pager so commands don't hang:

```bash
export GH_PAGER=cat
```

## Step 2: Fetch Repository and PR Info

```bash
REPO_PATH=$(gh repo view --json owner,name --jq '.owner.login + "/" + .name' | cat)
PR_ID=$(gh pr view --json number --jq '.number' | cat)
echo "Repository: $REPO_PATH  PR: $PR_ID"
```

## Step 3: Fetch All PR Review Comments and Threads

Always use `--paginate` to avoid missing comments due to page size limits.

```bash
# Inline review comments (tied to specific code lines)
gh api --paginate \
  -H "Accept: application/vnd.github+json" \
  -H "X-GitHub-Api-Version: 2022-11-28" \
  "/repos/$REPO_PATH/pulls/$PR_ID/comments" | cat

# Top-level review body comments (general feedback)
gh pr view --json reviews | cat
```

**Always pipe `gh` output to `cat`** — without it the pager blocks the command.

Additionally fetch the **review threads** via GraphQL. This is the authoritative source for whether a
thread is resolved (`isResolved`) and gives you the thread node ID (`PRRT_…`) required to resolve it in
Step 6. It also maps each thread to its first comment's `databaseId`, which you need to post replies.

Paginate with `pageInfo`/`endCursor` — a PR with more than 100 review threads will otherwise silently
drop the rest:

```bash
QUERY='
query($owner:String!,$name:String!,$pr:Int!,$after:String){
  repository(owner:$owner,name:$name){
    pullRequest(number:$pr){
      reviewThreads(first:100, after:$after){
        nodes{
          id
          isResolved
          comments(first:1){ nodes{ databaseId author{login} path body } }
        }
        pageInfo{ hasNextPage endCursor }
      }
    }
  }
}'

CURSOR=""
while :; do
  if [ -z "$CURSOR" ]; then
    PAGE=$(gh api graphql -f query="$QUERY" -F owner="${REPO_PATH%/*}" -F name="${REPO_PATH#*/}" -F pr="$PR_ID")
  else
    PAGE=$(gh api graphql -f query="$QUERY" -F owner="${REPO_PATH%/*}" -F name="${REPO_PATH#*/}" -F pr="$PR_ID" -F after="$CURSOR")
  fi
  echo "$PAGE" | cat
  HAS_NEXT=$(echo "$PAGE" | jq -r '.data.repository.pullRequest.reviewThreads.pageInfo.hasNextPage')
  [ "$HAS_NEXT" = "true" ] || break
  CURSOR=$(echo "$PAGE" | jq -r '.data.repository.pullRequest.reviewThreads.pageInfo.endCursor')
done
```

## Step 4: Analyze Comment Threads

Process **both** kinds of feedback, but they are analyzed differently:

- **Inline review threads** (has `path`/`line` from the REST API, appears in the GraphQL `reviewThreads`
  list) — use the GraphQL `isResolved` flag from Step 3 as the authoritative signal: skip any thread
  already `isResolved: true`. When you cannot rely on it (e.g. only the REST API is available via an
  MCP server that doesn't expose it), fall back to inferring resolution by content:
  - PR author acknowledged with valid reasoning for not changing
  - PR author explained an alternative approach
  - PR author responded substantively (even without reviewer follow-up)

- **Top-level review body comments** (from `gh pr view --json reviews`) — these are **not** part of
  `reviewThreads` and have no `isResolved` flag or thread ID to resolve. Judge resolution purely by
  content, the same way as the REST-API fallback above.

For every comment, regardless of kind, determine **actionability** — a comment is actionable unless it
is a purely informational acknowledgement:
- Comments prefixed with `d:` are still actionable
- LGTM comments that contain suggestions are actionable (e.g. "lgtm, but it would be useful to…")
- Filter out: resolved threads, pure acknowledgements

## Step 5: Interactive Comment Resolution

**Check for YOLO mode** — if the user included the word `yolo` in their initial request, activate YOLO mode immediately and share a pseudo-spiritual zen-buddhist koan about software engineering before processing.

Process **one comment at a time**:

### 5.1 Present the Comment

Show:
- **Location**: clickable VS Code link — `[filename:line](file:///absolute/path/to/file#line)` (construct from the repo root + `path` + `line` fields in the API response)
- Comment author
- Comment body
- Relevant thread context

### 5.2 Propose a Solution

If a solution is clear:
- Explain the issue
- Propose a specific implementation
- Show relevant code context if helpful
- **Ask for approval** with these options:
  - `fix` — implement the proposed solution
  - `skip` — skip this comment
  - `yolo` — fix this and all remaining comments automatically

If no solution is clear:
- Explain why it is difficult to address
- Ask the user for guidance, then move on

### 5.3 Wait for User Response

**Do NOT proceed without user input** (unless already in YOLO mode):

| Input | Action |
|-------|--------|
| `fix` | Implement, confirm briefly, next comment |
| `skip` | Move to next comment without changes |
| `yolo` | Enable YOLO mode, share a zen koan, implement current fix, continue all remaining comments automatically — skip any where solution is unclear |
| Custom guidance | Adjust solution and ask again |

### 5.4 Implement Changes

After approval (or in YOLO mode):
- Apply code changes
- Run relevant tests if applicable
- Confirm completion briefly
- Move to the next comment

## Step 6: Reply and Resolve Threads

For **every comment you addressed** (fixed, or consciously decided against with a rationale), post a short
reply describing what you did and then resolve the thread. Do this after the code changes are made — batch it
at the end of YOLO mode, or per-comment in interactive mode.

**Reply** to a thread with the REST replies endpoint, using the first comment's `databaseId` from Step 3:

```bash
gh api -X POST "/repos/$REPO_PATH/pulls/$PR_ID/comments/$COMMENT_DB_ID/replies" \
  -f body="Done — <one line describing the change>."
```

**Resolve** the thread with the GraphQL mutation, using the thread node ID (`PRRT_…`) from Step 3:

```bash
gh api graphql \
  -f query='mutation($id:ID!){ resolveReviewThread(input:{threadId:$id}){ thread{ id isResolved } } }' \
  -f id="$THREAD_ID"
```

Reply guidelines:
- Keep replies to one or two sentences: what changed and, if relevant, why. Reference the concrete
  construct you changed (method, flag, file) so the reviewer can verify without re-reading the diff.
- For `d:`/opinion comments you agree with, say so and describe the change. If you consciously decided
  *not* to change something, reply with the reasoning and still resolve — a resolved thread with a
  rationale is the record.
- **Only reply "done"/resolve for changes that are (or will be) pushed.** A resolved thread with no
  corresponding pushed code misleads reviewers. If nothing is pushed yet, either push first, or confirm
  with the user before resolving, and note in the reply that the change is pending push.
- Do **not** resolve threads owned by others that you did not act on, and do not resolve threads you
  skipped.

## Step 7: Completion

After all actionable comments are processed:
- List changes made with clickable VS Code links
- List any skipped comments (and why), and any threads left unresolved
- Confirm which threads were replied to and resolved
- Suggest next steps (commit, run full build/tests, push)

## Guidelines

This skill has two modes, and the strict rules below are what defines the difference between them:

- **Interactive mode (default):** address one comment at a time, and never implement a fix without the
  user's explicit approval for that comment.
- **YOLO mode** (user included `yolo`): address every actionable comment automatically without pausing
  for approval, and batch the replies/resolves at the end.

The remaining rules apply in both modes:

- Always reply to and resolve the threads you addressed (Step 6) — this is not optional
- Be honest: if a fix is unclear, say so
- Accept `skip` without argument
- Read the relevant file to understand context before proposing a fix
- If GitHub API fails, report clearly and continue to the next comment
