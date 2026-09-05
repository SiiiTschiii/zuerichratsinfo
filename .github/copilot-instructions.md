# AI Coding Agent Instructions

## Pull Requests

**Every description opens with `## What`, `## Why` and `## How`, in that order,
as top-level sections**, with nothing above them:

- **What** — the scope and nature of the change, in a sentence or two: what a
  caller, a user or an operator sees differently once this merges.
- **Why** — the problem, bug or missing behaviour that motivated the work, and
  the tradeoffs taken: the alternative rejected, the constraint that forced the
  design. Nobody can reconstruct this from the diff.
- **How** — the technical approach and architecture decisions, so a reviewer
  understands the solution before opening the diff. Name the files, functions
  and migrations that carry it.

Then `## Testing`, which is required rather than offered — commands run,
environment, edge cases covered and what was not. A change nobody exercised
beyond CI says so. Issue links go last; a closing keyword closes that issue on
merge, so only write one you mean.

Aim for 200–400 words, use bullets for multi-step logic rather than a paragraph
that has to be read twice, and write the body for `git log` in a year rather
than for the review thread: no "as requested", no per-push narration, no branch
commit hashes. Merges are squash-only and the body becomes the commit message
(`squash_merge_commit_message = PR_BODY`).

Full convention: [CLAUDE.md](../CLAUDE.md).
