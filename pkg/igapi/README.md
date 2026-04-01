# Instagram API Package

This package will implement a Go client for posting images to Instagram using the [Instagram Graph API](https://developers.facebook.com/docs/instagram-platform/instagram-api-with-facebook-login/content-publishing) via Facebook Login.

## Setup

### Prerequisites

- A **Facebook Page** for ZueriRatsinfo
- A **professional Instagram account** linked to the Facebook Page
- A **Meta Developer App** with the Instagram use case configured using **API setup with Facebook Login**
- Permissions: `instagram_basic`, `instagram_content_publish`, `pages_read_engagement`, `pages_show_list`

### Getting IDs

**Facebook Page ID** — call `GET /me/accounts` with a user access token:

```bash
curl "https://graph.facebook.com/v25.0/me/accounts?access_token=<USER_TOKEN>"
```

**Instagram User ID** — call `GET /{page-id}?fields=instagram_business_account`:

```bash
curl "https://graph.facebook.com/v25.0/<PAGE_ID>?fields=instagram_business_account&access_token=<USER_TOKEN>"
# → {"instagram_business_account": {"id": "<IG_USER_ID>"}, ...}
```

### Access Token

Generate a **User Access Token** in the [Graph API Explorer](https://developers.facebook.com/tools/explorer/) with the permissions listed above.

User tokens are short-lived (~1 hour in the Explorer, ~60 days when exchanged for a long-lived token). For automated use in GitHub Actions, exchange for a long-lived token and store as a secret (`IG_ACCESS_TOKEN`).

## API Flow

Publishing an image is a two-step process:

### Step 1 — Create a media container

```bash
curl -X POST "https://graph.facebook.com/v25.0/{ig-user-id}/media" \
  -d "image_url=https://example.com/image.jpg" \
  -d "caption=Your caption here" \
  -d "access_token=<TOKEN>"
# → {"id": "<CONTAINER_ID>"}
```

The image must be:

- Hosted on a **publicly accessible URL** (Meta fetches it directly)
- In **JPEG format** (only format supported for image posts)

### Step 2 — Publish the container

```bash
curl -X POST "https://graph.facebook.com/v25.0/{ig-user-id}/media_publish" \
  -d "creation_id=<CONTAINER_ID>" \
  -d "access_token=<TOKEN>"
# → {"id": "<MEDIA_ID>"}
```

The returned `id` is the published Instagram post ID.

## Rate Limits

- 100 API-published posts per 24-hour rolling period
- Check current usage: `GET /{ig-user-id}/content_publishing_limit`

## Troubleshooting

**Container status** — if `media_publish` doesn't return a media ID, check the container status:

```bash
curl "https://graph.facebook.com/v25.0/<CONTAINER_ID>?fields=status_code&access_token=<TOKEN>"
```

Possible values: `EXPIRED`, `ERROR`, `FINISHED`, `IN_PROGRESS`, `PUBLISHED`

**Page Publishing Authorization (PPA)** — if posting fails with an authorization error, the linked Facebook Page may require PPA. Complete it at [facebook.com/business](https://www.facebook.com/business/m/one-sheeters/page-publishing-authorization).
