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

### 1. Add `pages_manage_posts` to the token

Publishing to a Page feed needs `pages_manage_posts`. The Instagram token does
not carry it, so the Page token has to be regenerated with it.

**No App Review.** Meta requires App Review only for apps that reach data their
owner does not own or manage. The bot posts to its own Page with a token that
the app's own admin issued, so Standard Access is enough. That is the same
footing the Instagram permissions are on today. Review would only become
necessary if someone without a role on the app had to authorise it.

#### 1a. Allow the permission in the app

The Graph API Explorer offers only the permissions that the app's use cases
include.

1. Open <https://developers.facebook.com/apps/>, then the **ZueriRatsinfo** app.
2. In the left sidebar, open **Use cases**, then click **Customize** on the
   Instagram use case. This opens **Customize use case**, with **Instagram API**
   selected in the dropdown at the top of its own sidebar.
3. That sidebar has four entries. The bot uses **API setup with Facebook
   login**: its section 1 lists exactly the permissions the Instagram token
   already has (`instagram_basic`, `instagram_content_publishing`,
   `pages_read_engagement`, `business_management`, `pages_show_list`). Ignore
   **API setup with Instagram login** and **API integration helper**. Their
   `instagram_business_*` permissions and `graph.instagram.com` calls belong to
   the other login flow, so don't click **Add all required permissions** there.
4. Click **Permissions and features** in the sidebar, or the
   **Go to permissions and features** button on the Facebook-login page. Find
   `pages_manage_posts` in the list and click **Add**.

The ZueriRatsinfo app lists it there. If yours does not, click **Add more to
this use case** at the bottom of the sidebar and add **Manage everything on your Page**.
Then switch the dropdown to that use case and add `pages_manage_posts` under its
**Permissions and features** in the same way.

The Facebook-login page also has a **Complete app review** link, which offers
Advanced Access. Skip it, for the reason given above.

#### 1b. Regenerate the Page token

Follow [Access Token](../igapi/README.md#access-token-never-expiring-page-token)
in the Instagram setup, steps 1 to 5, with two differences:

- **Step 2:** the permission list there already includes `pages_manage_posts`.
  Make sure it is ticked in the Explorer before you click
  **Generate Access Token**. In the consent dialog, keep the ZueriRatsinfo Page
  selected and grant everything that is asked.
- **Step 5:** run the check below before storing the token.

The new token replaces `ZURICH_IG_ACCESS_TOKEN`. Instagram keeps working with
it, and the workflow hands the same secret to Facebook.

#### 1c. Check the token before storing it

Run this in the shell where you ran the Instagram steps. It needs `FB_APP_ID`
and `FB_APP_SECRET` (step 1), `FB_LONG_TOKEN` (step 3), `IG_TOKEN` (step 4) and
`FB_PAGE_ID` ([Getting IDs](../igapi/README.md#getting-ids)).

First check that the token carries the permission:

```bash
curl -s "https://graph.facebook.com/v25.0/debug_token?input_token=$IG_TOKEN&access_token=$FB_APP_ID|$FB_APP_SECRET" | jq '{type: .data.type, scopes: .data.scopes}'
```

The output should show `"type": "PAGE"`, and the scopes should include
`"pages_manage_posts"`. If the scope is missing, it was not ticked in step 2, or
1a was skipped.

Then check that your account may post on the Page:

```bash
curl -s "https://graph.facebook.com/v25.0/me/accounts?fields=id,name,tasks&access_token=$FB_LONG_TOKEN" | jq --arg page "$FB_PAGE_ID" '.data[] | select(.id == $page) | .tasks'
```

The list must contain `"CREATE_CONTENT"`. If it does not, your Page role is too
narrow: give yourself full control in the Page's settings under **Page access**.

If both checks pass, store the token with step 5 of the Instagram setup.

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
