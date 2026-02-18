---
description: 'Analyze and Address GitHub PR Review Comments Interactively'
agent: 'agent'
tools: ['search/changes', 'edit/editFiles', 'edit/createFile', 'search', 'execute/getTerminalOutput','execute/runInTerminal','read/terminalLastCommand','read/terminalSelection']
---

# PR Review Comment Analysis and Action

This prompt analyzes GitHub PR review comments and helps address them interactively.

## Overview

This workflow fetches PR review comments, analyzes conversation threads, filters out resolved or informational comments, and then addresses remaining actionable comments one by one in an interactive manner.

## Step 0: Configure Pager

**CRITICAL**: Before executing any `gh` commands, disable the pager to prevent commands from hanging:

```bash
PAGER=cat
export PAGER
```

Without this configuration, `gh` commands will use a pager by default, causing them to never exit and blocking the workflow.

## Step 1: Fetch Repository Information

First, determine the repository path and PR number:

```bash
# Get repository path (owner/name)
REPO_PATH=$(gh repo view --json owner,name --jq '.owner.login + "/" + .name' | cat)
echo "Repository: $REPO_PATH"

# Get PR number
PR_ID=$(gh pr view --json number --jq '.number' | cat)
echo "PR Number: $PR_ID"
```

## Step 2: Fetch PR Review Comments

**CRITICAL**: Always use `--paginate` to fetch all comments. The default page size is limited and will cause comments to be skipped.

Fetch all inline review comments from the PR:

```bash
# Fetch all inline review comments with automatic pagination
gh api --paginate \
  -H "Accept: application/vnd.github+json" \
  -H "X-GitHub-Api-Version: 2022-11-28" \
  "/repos/$REPO_PATH/pulls/$PR_ID/comments" | cat
```

Also fetch top-level review comments:

```bash
gh pr view --json reviews | cat
```

**Important**: Always pipe `gh` commands to `cat` (e.g., `| cat`). Without this, the output is opened in a pager, which causes the command to hang and never finish.

## Step 3: Analyze Comment Threads

For each comment, analyze:

1. **Comment Sources**:
   - **Inline review comments** from `/repos/$REPO_PATH/pulls/$PR_ID/comments` (specific code locations)
   - **Top-level review body comments** from `gh pr view --json reviews` (general feedback not tied to specific lines)
   - **CRITICAL**: Process BOTH types of comments - inline AND top-level review body comments
   - Top-level review comments often contain important actionable feedback (e.g., "please add a test for X")

2. **Thread Resolution**:
   - **Note**: GitHub's API does not expose whether a conversation/thread was explicitly marked as "resolved" via the UI
   - Therefore, we must analyze the conversation content to infer resolution status
   - Check if the comment has `in_reply_to_id` property set
   - Build conversation threads by linking comments via `id` and `in_reply_to_id`
   - Determine if a thread is resolved by analyzing the conversation:
     - PR author acknowledged and provided valid reasoning for not making changes
     - PR author explained they will handle it differently
     - PR author responded substantively to the comment (even without reviewer's follow-up)
   - Mark entire thread as resolved if the PR author has responded with reasoning

3. **Comment Purpose**:
   - **Important**: Comments prefixed with `d:` are still actionable and should be processed
   - The `d:` prefix typically indicates discussion points or suggestions for improvement
   - Only mark comments as non-actionable if they are purely informational acknowledgements
   - Comments suggesting improvements, raising concerns, or requesting changes are actionable regardless of prefix
   - **LGTM comments with additional suggestions** (e.g., "lgtm, but it would be useful to...") are ACTIONABLE

4. **Actionable Comments**:
   - Filter out resolved threads
   - Filter out purely informational/acknowledgement comments (not suggestions or improvement requests)
   - Extract remaining comments that require action from BOTH inline and top-level review comments
   - **Include all comments with suggestions, concerns, or improvement requests even if prefixed with `d:`**
   - **Include LGTM comments that contain actionable suggestions** (e.g., "lgtm, but perhaps it would also be useful to...")

## Step 4: Interactive Comment Resolution

**Check for YOLO Mode Activation:**
- If the user mentioned the term `yolo` in their initial request when starting this prompt, activate YOLO mode immediately
- When YOLO mode is activated (either initially or via user command), enlighten the developer with a pseudo-spiritual zen-buddhist style koan in the spirit of YOLO that relates to software engineering
- In YOLO mode, process all comments automatically without asking for approval

For each actionable comment (process ONE at a time):

### 4.1 Present the Comment

Show the user:
- **Comment location**: Format file paths as clickable VS Code links using the format `[filename:line](file:///absolute/path/to/file#line)` when exact file path and line number are available from the GitHub API. For inline review comments, the API provides `path`, `line`, and `side` fields. Extract the repository root path from the workspace and construct the absolute file path. Example: `[MeshLandingZone.kt:235](file:///home/user/workspace/repo/core/meshobjects/src/main/kotlin/io/meshcloud/meshobjects/objects/MeshLandingZone.kt#235)`
- Comment author
- Comment body
- Any context from the thread

### 4.2 Analyze and Propose Solution

**If you can determine a solution:**
- Explain the issue raised in the comment
- Propose a specific solution with implementation details
- Show relevant code context if applicable
- **ASK the user for approval** with these exact options:
  - `fix` - Implement the suggested solution
  - `skip` - Skip this comment and move to the next one
  - `yolo` - Fix this comment AND all remaining comments automatically without further interaction

**If you cannot determine a solution:**
- Be honest about the limitation
- Explain what makes it difficult to address
- Ask if the user has guidance
- Move to the next comment

### 4.3 Wait for User Response

**Do NOT proceed automatically.** Wait for user input (unless in YOLO mode):

- **If user responds with `fix`**: Implement the solution as proposed, then move to next comment
- **If user responds with `skip`**: Move to the next comment without changes
- **If user responds with `yolo`**:
  - Enable YOLO mode
  - Share a pseudo-spiritual zen-buddhist style koan in the spirit of YOLO that relates to software engineering
  - Implement the current solution
  - Continue to all remaining comments automatically
  - For each remaining comment: analyze, propose solution, and implement if feasible
  - Skip any comment where solution cannot be determined
  - Do NOT ask for approval on subsequent comments
- **If user provides alternative guidance**: Adjust solution and ask for confirmation again

### 4.4 Implement Changes (After Approval or in YOLO Mode)

When user approves with `fix` or `yolo`:
- Make the necessary code changes
- Run relevant tests if applicable
- Confirm completion briefly
- Move to the next comment
- **In YOLO mode**: Continue automatically without waiting for user input

## Step 5: Completion

After processing all actionable comments:
- Provide a summary of changes made with clickable VS Code file links in the format `[filename:line](file:///absolute/path#line)` for each change
- List any comments that were skipped
- Suggest next steps (e.g., run tests, push changes)

## Important Guidelines

1. **One Comment at a Time**: Never batch multiple comment fixes together (unless in YOLO mode)
2. **Always Ask First**: Never implement changes without explicit user approval (unless in YOLO mode)
3. **Be Honest**: If you don't know how to fix something, say so (skip in YOLO mode)
4. **Respect User Choice**: Accept `skip` without argument
5. **Maintain Context**: Keep track of which comments have been processed
6. **Thread Awareness**: Consider the full conversation context when analyzing resolution
7. **File Context**: Read relevant files to understand the comment in context
8. **YOLO Mode Behavior**: Once activated, process all remaining comments automatically with best-effort solutions

## Expected User Commands

- `fix` - Proceed with your proposed solution for this comment only
- `skip` - Skip this comment, move to next
- `yolo` - Fix this and ALL remaining comments automatically without further interaction
- Custom guidance - Adjust solution based on user input

## Error Handling

- If GitHub API fails, report the error clearly
- If comment context is unclear, ask for clarification
- If file paths don't exist, report and skip
- If tests fail after changes, report and ask for guidance
