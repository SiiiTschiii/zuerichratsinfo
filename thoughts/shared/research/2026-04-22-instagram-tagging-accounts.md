---
date: 2026-04-22T10:30:00+00:00
researcher: copilot
topic: "Tagging accounts via Instagram Graph API"
tags: [research, instagram, graph-api, tagging]
status: complete
last_updated: 2026-04-22
---

# Research: tagging accounts via Instagram Graph API

**Date**: 2026-04-22

## Research Question

Does the Instagram Graph API support tagging users in a feed/carousel post caption or in an auto-posted first comment, and should ZueriRatsinfo add mapped contact tagging for Instagram similar to X/Bluesky?

## Summary

Yes for captions: the IG media creation endpoint supports `caption` with `@username` mentions, and Meta documents that mentioned users are notified.

Yes for auto-posted first comments: after publishing media, IG comments can be created via `POST /<IG_MEDIA_ID>/comments?message=...`, which enables a programmatic “first comment”.

Given this, we should tag mapped contacts in the Instagram caption (equivalent to X/Bluesky name-based mapping). We do **not** add first-comment posting in this change because that would require additional posting logic and `instagram_manage_comments` permissions.

## Detailed Findings

### 1. Caption mentions are supported

Meta’s IG User media reference (`/<IG_USER_ID>/media`) documents:

- `caption` can include Instagram usernames such as `@natgeo`
- `@`-mentioned users receive a notification when the container is published
- caption limits include max 2200 chars and max 20 `@` tags

This directly supports our existing caption-based Instagram publishing path.

### 2. “First comment” can be auto-posted via API

Meta’s IG Media comments reference documents:

- `POST /<IG_MEDIA_ID>/comments?message=<MESSAGE_CONTENT>`
- this creates a comment on a published media object

So an automated first comment is possible after publish if desired.

### 3. Current repository integration point

The current Instagram flow in this repo already builds and posts a caption:

- caption formatting: `pkg/voteposting/platforms/instagram/format.go`
- media creation/publish flow: `pkg/igapi/client.go` and `pkg/voteposting/platforms/instagram/platform.go`

Therefore caption tagging is the minimal, low-risk way to add mapped Instagram tagging now.

## Decision

Implement mapped-contact Instagram tagging in caption text (name match → `@instagram_handle`) and keep first-comment posting out of scope for now.

## Sources

- Meta docs mirror of `IG User Media` with source URL `https://developers.facebook.com/docs/instagram-platform/instagram-graph-api/reference/ig-user/media` (retrieved copy includes `caption` + mention behavior)
- Meta docs mirror of `IG Media Comments` with source URL `https://developers.facebook.com/docs/instagram-platform/instagram-graph-api/reference/ig-media/comments` (documents `POST /<IG_MEDIA_ID>/comments`)
- Repository code:
  - `pkg/voteposting/platforms/instagram/format.go`
  - `pkg/voteposting/platforms/instagram/platform.go`
  - `pkg/igapi/client.go`
