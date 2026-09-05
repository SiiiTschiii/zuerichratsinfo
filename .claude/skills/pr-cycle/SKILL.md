---
name: pr-cycle
description: Run the one Copilot review pass a pull request gets before Christof sees it — wait for CI, address and resolve the review comments, refresh the description. Use after opening a PR, after pushing to a branch with an open PR, or when asked about a PR's review state.
---

# One review pass on a pull request

PR number from the argument, else `gh pr view --json number -q .number`. `$REPO`
is `gh repo view --json nameWithOwner -q .nameWithOwner`.

## 1. Request the review

First, so it runs while CI does:

```bash
gh pr edit <pr> --add-reviewer "@copilot"
```

Once per PR, ever. After pushing fixes, do not ask again — the next read is
Christof's.

## 2. Wait

```bash
.claude/scripts/pr-await.sh <pr>                # opening pass
.claude/scripts/pr-await.sh <pr> --checks-only  # every later push
```

**In the background.** Prints check conclusions, bot comments, and unresolved
threads with their GraphQL ids.

Omitting `--checks-only` on a later push waits for a review that is never coming
and burns the full timeout. Exit 2: someone pushed, rerun on the new head. Exit
3: timeout, report what is pending.

## 3. Read it

Red checks first; a failing branch is not worth reviewing.

Every test on the branch, yours or Copilot's, answers **what change to the
source would make this fail?** before it stays. No answer means it goes. A
review comment asking for a test that would pin a boundary, a validation rule or
a shape decision is worth implementing; one asking for a test because a file is
uncovered is argued down rather than obeyed. Copilot readily writes the shapes
that cannot fail — an assertion that removed text is absent, a render with
nothing asserted after it, a snapshot locking in a diff — so read the tests in
its pull requests as carefully as the code.

## 4. Address it

Per unresolved thread:

- **Agree, unambiguous** → implement. Reasoning goes in the commit message, not
  the reply. One commit per point, so the hash names the change.
- **Disagree** → reply why. Copilot is often right about the code and wrong
  about this repo's conventions — check [CLAUDE.md](../../../CLAUDE.md) and the
  surrounding code before conceding.
- **Needs Christof** → reply naming the question, leave open, list in the handoff.

Push, then reply to each thread's first comment by `databaseId` so the hash
exists:

```bash
gh api repos/$REPO/pulls/<pr>/comments/<comment-id>/replies -f body="Fixed in <sha>. <one line>"
```

Resolve everything needing nothing further, by thread `id`:

```bash
gh api graphql -f query='mutation($id:ID!){resolveReviewThread(input:{threadId:$id}){thread{isResolved}}}' -F id=<thread-id>
```

## 5. Refresh the description

After *any* code change on the branch. It states the change as it would merge
right now — rules in [CLAUDE.md](../../../CLAUDE.md), "Pull request
descriptions". Also: no `closes #nn` unless you mean to close that issue.

Read the body back before pushing it. It opens with `## What`, `## Why` and
`## How`, in that order and with nothing above them, then `## Testing`; it is
200–400 words; and nothing in it is addressed to the reviewer rather than to
someone reading `git log`.

```bash
gh pr edit <pr> --body-file <file>
```

## 6. Hand off

One short message: check status, what you implemented, what you pushed back on,
what is left open for Christof.
