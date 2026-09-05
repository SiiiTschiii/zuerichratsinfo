#!/usr/bin/env bash
# Injects the pr-cycle reminder into the agent's context after it opens a pull
# request or pushes to a branch that has one. Wired to PostToolUse in
# .claude/settings.json; the procedure itself lives in .claude/skills/pr-cycle.
#
# Silence means no reminder: a push on a branch with no open PR, or no gh, is
# not an error here.
set -uo pipefail

mode=${1:-opened}

pr=$(gh pr view --json number,state -q 'select(.state == "OPEN") | .number' 2>/dev/null) || exit 0
[ -n "$pr" ] || exit 0

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
