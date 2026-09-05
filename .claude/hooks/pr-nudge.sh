#!/usr/bin/env bash
# Injects the pr-cycle reminder into the agent's context after it opens a pull
# request or pushes to a branch that has one. Wired to PostToolUse in
# .claude/settings.json; the procedure itself lives in .claude/skills/pr-cycle.
#
# Silence means no reminder: a push on a branch with no open PR, or no gh, is
# not an error here.
set -uo pipefail

# The mode is decided from the command rather than by a matcher in
# settings.json. That pattern is fuzzy — `git status` matches `git push *` —
# and a reminder delivered on an unrelated command is noise the agent carries
# for the rest of the session.
payload=""
[ -t 0 ] || payload=$(cat)
cmd=$(printf '%s' "$payload" | jq -r '.tool_input.command // ""' 2>/dev/null)

mode=${1:-}
case "$cmd" in
  *--dry-run*)      exit 0 ;;
  *"gh pr create"*) mode=opened ;;
  *"git push"*)     mode=pushed ;;
esac
[ -n "$mode" ] || exit 0

pr=$(gh pr view --json number,state -q 'select(.state == "OPEN") | .number' 2>/dev/null) || exit 0
[ -n "$pr" ] || exit 0

# One reminder per head commit per mode. Without this a re-push of an unchanged
# branch, or a matcher that fires twice for one tool call, repeats a paragraph
# the agent has already read.
if git_dir=$(git rev-parse --git-dir 2>/dev/null) && head=$(git rev-parse HEAD 2>/dev/null); then
  marker="$git_dir/pr-nudge-$pr-$mode-$head"
  [ -e "$marker" ] && exit 0
  : > "$marker"
fi

case "$mode" in
  opened)
    msg="Pull request #$pr is open. Follow .claude/skills/pr-cycle/SKILL.md (the /pr-cycle skill) now and run it to completion before handing back: request the Copilot review, wait with .claude/scripts/pr-await.sh in the background, then act on CI and the review comments."
    ;;
  pushed)
    msg="You pushed to the branch of pull request #$pr. Two obligations: the description must still describe the change as it would merge right now and still have the shape step 5 of .claude/skills/pr-cycle/SKILL.md requires — read it back against that step and update it if not — and the new head commit needs its CI watched, so run '.claude/scripts/pr-await.sh $pr --checks-only' in the background and fix anything red. Do NOT request another Copilot review; a pull request gets exactly one, at opening. If you are already partway through .claude/skills/pr-cycle/SKILL.md, continue it rather than restarting."
    ;;
  *) exit 0 ;;
esac

jq -n --arg c "$msg" \
  '{hookSpecificOutput:{hookEventName:"PostToolUse",additionalContext:$c}}'
