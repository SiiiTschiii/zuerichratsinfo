# Instagram API Package

This package implements a Go client for posting images to Instagram using the [Instagram Graph API](https://developers.facebook.com/docs/instagram-platform/instagram-api-with-facebook-login/content-publishing) via Facebook Login.

## Packages

- **`igapi.Client`** — Instagram Graph API client for carousel publishing (create containers, publish, poll status)
- **`igapi.ImageHoster`** — GitHub Pages image hosting via the GitHub Contents API (upload/cleanup JPEG files on `gh-pages` branch)

## Setup

### Prerequisites

- A **Facebook Page** for ZueriRatsinfo
- A **professional Instagram account** linked to the Facebook Page
- A **Meta Developer App** with the Instagram use case configured using **API setup with Facebook Login**
- Permissions: `instagram_basic`, `instagram_content_publish`, `pages_read_engagement`, `pages_show_list`
- A **GitHub repository** with a `gh-pages` branch for image hosting

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

### Environment Variables

| Variable | Description |
|---|---|
| `IG_USER_ID` | Instagram professional account ID |
| `IG_ACCESS_TOKEN` | Long-lived Page access token |
| `GITHUB_TOKEN` | GitHub token for pushing to gh-pages (available in Actions) |
| `IG_REPO_OWNER` | GitHub repository owner (for image hosting) |
| `IG_REPO_NAME` | GitHub repository name (for image hosting) |

## API Flow

### Carousel Publishing

Publishing a carousel is a multi-step process:

#### Step 1 — Upload images to GitHub Pages

Images are committed to the `gh-pages` branch under `ig-images/` using the GitHub Contents API. After a short delay for GitHub Pages deployment, the images become publicly accessible.

#### Step 2 — Create carousel item containers

For each image, create a media container with `is_carousel_item=true`:

```bash
curl -X POST "https://graph.facebook.com/v25.0/{ig-user-id}/media" \
  -d "image_url=https://example.github.io/repo/ig-images/img.jpg" \
  -d "is_carousel_item=true" \
  -d "access_token=<TOKEN>"
# → {"id": "<CONTAINER_ID>"}
```

#### Step 3 — Create carousel container

Combine all child containers into a carousel with a caption:

```bash
curl -X POST "https://graph.facebook.com/v25.0/{ig-user-id}/media" \
  -d "media_type=CAROUSEL" \
  -d 'children=["<CHILD_1>","<CHILD_2>"]' \
  -d "caption=Your caption here" \
  -d "access_token=<TOKEN>"
# → {"id": "<CAROUSEL_ID>"}
```

#### Step 4 — Publish the carousel

```bash
curl -X POST "https://graph.facebook.com/v25.0/{ig-user-id}/media_publish" \
  -d "creation_id=<CAROUSEL_ID>" \
  -d "access_token=<TOKEN>"
# → {"id": "<MEDIA_ID>"}
```

#### Step 5 — Poll for PUBLISHED status

```bash
curl "https://graph.facebook.com/v25.0/<CAROUSEL_ID>?fields=status_code&access_token=<TOKEN>"
```

Possible values: `EXPIRED`, `ERROR`, `FINISHED`, `IN_PROGRESS`, `PUBLISHED`

#### Step 6 — Clean up hosted images

After successful publishing, the hosted images are removed from the `gh-pages` branch.

## Rate Limits

- 100 API-published posts per 24-hour rolling period
- Check current usage: `GET /{ig-user-id}/content_publishing_limit`

## Troubleshooting

**Container status** — if `media_publish` doesn't return a media ID, check the container status:

```bash
curl "https://graph.facebook.com/v25.0/<CONTAINER_ID>?fields=status_code&access_token=<TOKEN>"
```

**Page Publishing Authorization (PPA)** — if posting fails with an authorization error, the linked Facebook Page may require PPA. Complete it at [facebook.com/business](https://www.facebook.com/business/m/one-sheeters/page-publishing-authorization).
