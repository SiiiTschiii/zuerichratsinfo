# Working in this repository

## Pull request descriptions

**Merges are squash-only, and the description becomes the commit message.** The
repository sets `squash_merge_commit_message = PR_BODY`, so whatever is in the
body is what `git log` shows for as long as the project exists — on a commit
whose hash did not exist when the body was written. Two consequences: never
point at a commit hash from the branch, because it is unreachable after the
squash; and write for someone reading `git log` in a year rather than for the
reviewer this week — no "as requested", no narration of what changed between
pushes.

### Every description opens with What, Why and How

Three top-level sections, in that order, on every pull request — no preamble
above them and nothing that pushes them below the fold. A reader who stops after
those three has the change, the reason for it and the shape of the solution.

- **`## What`** — the scope and nature of the change, in a sentence or two: what
  a caller, a user or an operator sees differently once this merges.
- **`## Why`** — the problem, bug or missing behaviour that made the work
  necessary, and the tradeoffs taken along the way: the alternative rejected, the
  constraint that forced the design. This is the part a reader cannot
  reconstruct from the diff, so it is the part worth the most words.
- **`## How`** — the technical approach and the architecture decisions, so a
  reviewer understands the solution before opening the diff. Name the files,
  functions and migrations that carry it.

Then `## Testing`, which is required rather than offered: how the change was
verified — commands run, environment, the edge cases covered, and honestly what
was not. A change nobody exercised beyond CI says so; that is information, and
its absence reads as an oversight. Anything a deployer has to do by hand gets
its own section, written as instructions rather than as a diary. Issue and
ticket links go last, on their own line (`Fixes #12`, `Refs #34`) — and a
closing keyword closes that issue on merge, so only write one you mean.

**200 to 400 words.** Past that the decisions a reader came for are buried, and
it is usually "How" that has overrun — the diff already says which lines moved.
Break multi-step logic into bullets rather than a paragraph that has to be read
twice.

## Opening a pull request

**Every pull request gets one machine review pass before a human sees it.** The
procedure is [.claude/skills/pr-cycle](.claude/skills/pr-cycle/SKILL.md): request
the Copilot review, wait for CI and that review, implement or argue down each of
its comments, resolve the threads that need nothing further, and bring the
description back in line with the code.

A `PostToolUse` hook in [.claude/settings.json](.claude/settings.json) starts the
cycle on `gh pr create`. On every later push it re-checks CI and the description
only — a pull request gets one Copilot review, at opening; the next read is
Christof's. The skill is the contract, though, not the hook: run `/pr-cycle` by
hand for a pull request that arrived some other way.
