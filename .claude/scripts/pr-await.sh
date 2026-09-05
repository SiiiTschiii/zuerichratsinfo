#!/usr/bin/env bash
# Blocks until a pull request is ready to act on: every check run on its head
# commit has finished, and — on the opening pass — Copilot has reviewed that
# same commit.
#
#   .claude/scripts/pr-await.sh <pr-number>                # opening pass
#   .claude/scripts/pr-await.sh <pr-number> --checks-only   # after a push
#
# Copilot reviews a pull request once, when it is asked to. Waiting for a second
# review after a later push would block until the timeout, so every pass after
# the first waits on CI alone.
#
# Both waits are pinned to the head sha the PR had when this started: CI results
# for an older commit say nothing about the code that would merge now.
#
# Run it in the background and read the summary it prints on exit.
# Exit 0: ready. 2: head moved. 3: timed out. 1: usage or API failure.
set -euo pipefail

pr=${1:?usage: pr-await.sh <pr-number> [--checks-only]}
want_review=1
[ "${2:-}" = "--checks-only" ] && want_review=0
timeout=${PR_AWAIT_TIMEOUT:-2400}
interval=${PR_AWAIT_INTERVAL:-20}
grace=${PR_AWAIT_GRACE:-90}
reviewer=copilot-pull-request-reviewer

repo=$(gh repo view --json nameWithOwner -q .nameWithOwner)
head=$(gh pr view "$pr" --json headRefOid -q .headRefOid)
deadline=$(( $(date +%s) + timeout ))
grace_until=$(( $(date +%s) + grace ))

echo "PR #$pr in $repo, head $head"

# Prints "<total> <pending>" for the head commit's check runs.
checks_state() {
  gh api "repos/$repo/commits/$head/check-runs" --paginate \
    -q '[(.check_runs | length), ([.check_runs[] | select(.status != "completed")] | length)] | @tsv' |
    awk '{t+=$1; p+=$2} END{print t+0, p+0}'
}

# A review counts only if it was submitted against the head commit, so a review
# left over from a commit that has since been amended does not end the wait.
copilot_reviewed() {
  gh api "repos/$repo/pulls/$pr/reviews" --paginate \
    -q "[.[] | select((.user.login | startswith(\"$reviewer\")) and .commit_id == \"$head\")] | length" |
    awk '{s+=$1} END{print s+0}'
}

while :; do
  now=$(gh pr view "$pr" --json headRefOid -q .headRefOid)
  if [ "$now" != "$head" ]; then
    echo "head moved $head -> $now; rerun against the new commit"
    exit 2
  fi

  read -r total pending <<<"$(checks_state)"
  reviewed=0
  [ "$want_review" -eq 1 ] && reviewed=$(copilot_reviewed)

  # An empty check-run set is ambiguous: Actions may not have registered the
  # workflows for a just-pushed sha yet, or the change may match no path filter
  # at all. Only an empty set that is still empty after the grace period is
  # taken as the second.
  checks_done=0
  if [ "$total" -eq 0 ]; then
    [ "$(date +%s)" -ge "$grace_until" ] && checks_done=1
  elif [ "$pending" -eq 0 ]; then
    checks_done=1
  fi

  if [ "$checks_done" -eq 1 ]; then
    [ "$want_review" -eq 0 ] && break
    [ "$reviewed" -gt 0 ] && break
  fi

  if [ "$(date +%s)" -ge "$deadline" ]; then
    echo "timed out after ${timeout}s: $total check(s), $pending running, $reviewed Copilot review(s)"
    exit 3
  fi
  sleep "$interval"
done

echo
echo "== checks =="
gh api "repos/$repo/commits/$head/check-runs" --paginate \
  -q '.check_runs[] | "\(.conclusion // "?")\t\(.name)"' | sort

echo
echo "== bot comments on the PR (coverage and the like) =="
gh api "repos/$repo/issues/$pr/comments" --paginate \
  -q '.[] | select(.user.type == "Bot") | "--- \(.user.login) ---\n\(.body)"'

echo
echo "== unresolved review threads =="
gh api graphql --paginate -f query='
query($owner:String!,$repo:String!,$number:Int!,$endCursor:String){
  repository(owner:$owner,name:$repo){
    pullRequest(number:$number){
      reviewThreads(first:100, after:$endCursor){
        pageInfo{hasNextPage endCursor}
        nodes{
          id isResolved isOutdated
          comments(first:1){nodes{databaseId author{login} path line originalLine body}}
        }
      }
    }
  }
}' -F owner="${repo%%/*}" -F repo="${repo##*/}" -F number="$pr" \
  -q '.data.repository.pullRequest.reviewThreads.nodes[]
      | select(.isResolved | not)
      | .comments.nodes[0] as $c
      | "\(.id)\t\($c.author.login)\t\($c.path):\($c.line // $c.originalLine // 0)\t\($c.body | split("\n")[0])"'
