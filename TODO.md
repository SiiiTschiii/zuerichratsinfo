# Project Roadmap

- [x] Initial setup
  - Create GitHub repo
  - Set up Go project structure
  - First version that fetches data from PARIS API and posts to X
- [x] Automate with GitHub Actions
  - Set up a GitHub Actions workflow to run the bot on a schedule (e.g., hourly or daily) and post to X automatically.
- [x] Define first real use case
  - Decide what type of council business to post (e.g., only Motions, or all new submissions) and when to post (immediately, daily summary, etc.).
- [x] Design a Logo for the x.com account
- [x] Write a transparent X account description
- [x] use verified account, also to be ablo to post more than 280 characters if needed.
- [x] post all new abstimmungen. how to avoid duplicates? keep track of a published abstimmuneng log. In the repo with the github action that commits to it after it run?
- [x] Tag relevant X accounts. Enhance the bot to tag relevant Gemeinderat X accounts in posts, based on author/submitter.
- [x] Curate the contacts.yaml with social media accounts.
- [x] Contact politicians on x.com and some news orgs to follow and share the account.
- [x] Add a way to suppport the project (e.g., GitHub Sponsors, Buy Me a Coffee, etc.). The x.com premium accounts / paid API features.
- [x] Add a thank you section in the README to acknowledge contributors and supporters.
- [x] Group Abstimmungen with the same date and Gschäft into one single post
- [x] Based on the contacts.yaml, decide which social media platforms to expand to next (Instagram, Facebook, LinkedIn, TikTok, Bluesky)
  - Created `cmd/platform_stats` to analyze platform presence
  - Statistics displayed in README.md "Supported Platforms" table
  - GitHub Action automatically updates README on contacts.yaml changes
  - Decision: Bluesky → LinkedIn → Facebook → Instagram → TikTok (by effort/fit)
- [ ] Generate visual posts (images)
  - Create simple image posts with text on colored backgrounds (varying colors per post, add shadows)
  - This would enable expansion to visual-first platforms like Instagram and TikTok
  - Start simple: uni-colored background + large text + basic shadow effects

## Platform Integrations

### Bluesky (Priority 1 — lowest effort, closest to X format)

- [ ] Implement Bluesky client using AT Protocol (`com.atproto.repo.createRecord`)
  - Auth via username/password session, no app review needed
  - Docs: https://docs.bsky.app/docs/get-started
- [ ] Implement Bluesky post formatter (300 char limit, link facets, optional image embed)
  - Adapt X format: emoji header, result, vote counts, shortened link
  - Docs: https://docs.bsky.app/docs/advanced-guides/posts
- [ ] Add Bluesky platform to posting pipeline and track posted votes separately

### LinkedIn (Priority 2 — professional audience, text posts work)

- [ ] Implement LinkedIn client using Share on LinkedIn (Consumer API, `POST /v2/ugcPosts`)
  - OAuth 2.0 with `w_member_social` scope, add "Share on LinkedIn" product in Dev Portal
  - Docs: https://learn.microsoft.com/en-us/linkedin/consumer/integrations/self-serve/share-on-linkedin
- [ ] Implement LinkedIn post formatter (3,000 char limit, article/URL shares with link preview)
  - Longer format: full title, vote counts, hashtags (#GemeinderatZürich #Abstimmung)
- [ ] Add LinkedIn platform to posting pipeline and track posted votes separately

### Facebook (Priority 3 — wide reach, requires Meta App Review)

- [ ] Register Facebook App, create Page, and go through App Review for `pages_manage_posts`
  - Docs: https://developers.facebook.com/docs/pages-api/posts/
- [ ] Implement Facebook client using Pages API (`POST /{page-id}/feed`)
  - Text + link posts; Facebook auto-generates link preview cards
- [ ] Implement Facebook post formatter and add to posting pipeline

### Instagram (Priority 4 — visual-first, requires image generation)

- [ ] Implement vote result image generator (infographic: title, bar chart, result)
  - Prerequisite: "Generate visual posts" TODO above
  - Docs: https://developers.facebook.com/docs/instagram-platform/instagram-api-with-instagram-login/content-publishing
- [ ] Implement Instagram client using Content Publishing API (image + caption)
  - Requires professional IG account + Facebook Page + App Review; 100 posts/24h limit
- [ ] Add Instagram platform to posting pipeline and track posted votes separately

### TikTok (Priority 5 — highest effort, video/photo required, audit needed)

- [ ] Register TikTok Developer App, enable Content Posting API, get `video.publish` scope approved
  - All posts are private until app passes TikTok audit
  - Docs: https://developers.tiktok.com/doc/content-posting-api-get-started/
- [ ] Implement TikTok client using Content Posting API (photo or video post)
  - Photo post via `POST /v2/post/publish/content/init/`; images must be on verified domain
- [ ] Implement TikTok post formatter (photo/video + caption) and add to pipeline
