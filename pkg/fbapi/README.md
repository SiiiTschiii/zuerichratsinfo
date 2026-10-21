# Facebook Pages API Package

This package publishes text posts with a link to the ZueriRatsinfo Facebook Page
through the [Pages API](https://developers.facebook.com/docs/pages-api/posts/):
`POST /{page-id}/feed` with a `message` and a `link`. Facebook scrapes the link
and attaches the preview card itself, so the bot generates and hosts no images
for this platform.

## Setup

Facebook rides on the Instagram setup: the same Meta Developer App, the same
Facebook Page and the same never-expiring Page access token. Do
[the Instagram setup](../igapi/README.md) first; this section covers only what
Facebook adds.

### 1. Grant `pages_manage_posts`

Publishing to a Page feed needs `pages_manage_posts`, which the Instagram token
does not carry. Add it to the app's App Review request alongside the Instagram
permissions, then regenerate the Page token following
[the Instagram token steps](../igapi/README.md#access-token-never-expiring-page-token),
ticking `pages_manage_posts` in step 2. The regenerated token replaces
`ZURICH_IG_ACCESS_TOKEN`; Instagram keeps working with it.

Confirm the new token carries the permission (`FB_APP_ID` and `FB_APP_SECRET`
as exported in the Instagram steps, `IG_TOKEN` the regenerated Page token):

```bash
curl -s "https://graph.facebook.com/v25.0/debug_token?input_token=$IG_TOKEN&access_token=$FB_APP_ID|$FB_APP_SECRET" | jq '.data.scopes'
```

The list must contain `"pages_manage_posts"`.

### 2. Seed the vote log

A platform whose vote log is missing treats every vote inside
`MAX_VOTE_AGE_DAYS_*` as unposted — 90 days for Stadt Zürich — and would work
through that backlog at `ZURICH_FB_MAX_POSTS_PER_RUN` posts an hour. Copy the
Instagram logs so the Page starts from the votes still to come. Pause the bot
first, then wait for every queued and running run to finish — disabling stops
new triggers but not those, and a run whose push lands after the seed push is
rejected and loses its record of what it posted. Do not cancel them instead: a
run may already have posted.

```bash
gh workflow disable bot.yml
while gh run list --workflow bot.yml --limit 20 --json status -q '.[].status' | grep -qv completed; do sleep 30; done
git fetch origin state-log
git worktree add --detach ../state-log-seed origin/state-log
for j in zurich-city zurich-canton; do
  jq '.platform = "facebook"' "../state-log-seed/data/$j/posted_votes_instagram.json" > "../state-log-seed/data/$j/posted_votes_facebook.json"
done
git -C ../state-log-seed add data
git -C ../state-log-seed commit -m "Seed Facebook vote logs from Instagram [skip ci]"
git -C ../state-log-seed push origin HEAD:state-log
git worktree remove ../state-log-seed
gh workflow enable bot.yml
```

### 3. Switch it on

The Page ID is not a secret and is what turns Facebook on — the workflow passes
the Instagram token through under `ZURICH_FB_ACCESS_TOKEN`. With `FB_PAGE_ID` as
found under [Getting IDs](../igapi/README.md#getting-ids):

```bash
gh variable set ZURICH_FB_PAGE_ID --body "$FB_PAGE_ID"
```

Unset it to stop Facebook without touching the other platforms:

```bash
gh variable delete ZURICH_FB_PAGE_ID
```

### Environment Variables

Names are channel-scoped — the `zurich` channel reads `ZURICH_FB_PAGE_ID`.

| Variable                         | Description                                          |
| -------------------------------- | ---------------------------------------------------- |
| `<CHANNEL>_FB_PAGE_ID`           | Facebook Page ID                                     |
| `<CHANNEL>_FB_ACCESS_TOKEN`      | Page access token with `pages_manage_posts`          |
| `<CHANNEL>_FB_MAX_POSTS_PER_RUN` | Posts per hourly run (default 5)                     |

## Trying it locally

Preview what the bot would post, without credentials:

```bash
go run ./cmd/generate_vote_post -platform facebook -n 2
```

Publish one test fixture to the real Page:

```bash
ZURICH_FB_PAGE_ID=$FB_PAGE_ID ZURICH_FB_ACCESS_TOKEN=$IG_TOKEN go run ./cmd/post_fixture -platform facebook -fixture single-vote-angenommen
```

The output ends with the post URL. Delete the post from the Page afterwards.
