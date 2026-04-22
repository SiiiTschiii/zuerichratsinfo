---
date: 2026-04-22T10:32:28+0000
researcher: copilot
topic: "Per-platform campaign email sending (Instagram, Facebook, TikTok, LinkedIn following the Bluesky precedent)"
tags: [research, codebase, send_campaign_emails, contacts, instagram, bluesky]
status: complete
last_updated: 2026-04-22
---

# Research: Campaign emails for Instagram (and future Facebook / TikTok / LinkedIn) contacts

**Date**: 2026-04-22

## Research Question

The user previously ran a campaign (`cmd/send_campaign_emails`) that emailed every contact who has a Bluesky account to announce the new Bluesky presence. Now that Instagram posting is live, they want to run the equivalent campaign for every contact with an Instagram account, and eventually do the same for Facebook, TikTok, and LinkedIn. This document maps out what exists today in the codebase that supports that workflow.

## Summary

- The campaign mailer lives at [cmd/send_campaign_emails/main.go](cmd/send_campaign_emails/main.go) and is currently **hard‑coded to Bluesky** in four concrete places: the subject constant, the filter over `contact.Bluesky`, the `Recipient.BlueskyURL` field, and two Bluesky‑specific email body templates (verify/preview vs. real send).
- Contact data already supports all six platforms uniformly. The `contacts.Contact` struct in [pkg/contacts/contacts.go](pkg/contacts/contacts.go) has `X`, `Facebook`, `Instagram`, `LinkedIn`, `Bluesky`, `TikTok` fields, and `Mapper.GetPlatformURLs(name, platform)` already switches over all six platform names.
- The Instagram project account is **`@zueriratsinfo`** (`https://www.instagram.com/zueriratsinfo`) per [README.md](README.md) line 32 — note the spelling differs from the Bluesky handle `zuerichratsinfo.bsky.social`.
- Coverage numbers from [README.md](README.md): Instagram 99, Facebook 102, TikTok 21, LinkedIn 121, Bluesky 36, X 74 contacts.
- All non‑platform pieces of the pipeline (API lookup for emails, gender→salutation, overrides file, preview/test/send modes, SMTP code) are platform‑agnostic and reusable as‑is.

## Detailed Findings

### Campaign mailer entry point

[cmd/send_campaign_emails/main.go](cmd/send_campaign_emails/main.go)

- Flags: `--contacts`, `--overrides`, `--preview`, `--output`, `--test <addr>`, `--send`, `--delay` (main.go:19-27).
- Four modes dispatched from `main()` (main.go:49-68): default = `runVerify` (table), `runPreview` (markdown dump), `runSend` with `--test` override, `runSend` real.
- `main.go:29` hard‑codes `const emailSubject = "zuerichratsinfo jetzt auch auf Bluesky"`.

### Recipient struct (Bluesky‑specific field)

[cmd/send_campaign_emails/main.go:31-38](cmd/send_campaign_emails/main.go)

```go
type Recipient struct {
    Name       string
    Email      string
    Gender     string
    Salutation string
    BlueskyURL string
    Source     string
}
```

Only `BlueskyURL` is platform‑specific; the rest are generic.

### Contact filtering (Bluesky‑only)

[cmd/send_campaign_emails/main.go:76-83](cmd/send_campaign_emails/main.go)

```go
allContacts := mapper.GetAllContacts()
var bskyContacts []contacts.Contact
for _, c := range allContacts {
    if len(c.Bluesky) > 0 {
        bskyContacts = append(bskyContacts, c)
    }
}
```

Then `main.go:105` picks the first URL: `bskyURL := contact.Bluesky[0]`. This is the only place the campaign command reads the contact's social fields.

### Email lookup via PARIS API + overrides

- `zurichapi.NewClient().FetchAllKontakte()` returns the city's official contact records (main.go:88-94).
- `findEmailForContact` (main.go:183-213) fuzzy‑matches by splitting the contact name on whitespace and requiring every token to be a substring of `"{Name} {Vorname}"` from the API; returns `EmailPrivat` if set, else `EmailGeschaeft`, plus `Geschlecht`.
- `data/email_overrides.yaml` provides manual overrides keyed by `name` (matched case‑insensitively) with `email` and `gender`; see [data/email_overrides.yaml](data/email_overrides.yaml) (6 entries as of this commit). Overrides take priority over API data (main.go:101-118).
- Gender → salutation via [main.go:158-163](cmd/send_campaign_emails/main.go): `"weiblich"` → `"Liebe"`, else `"Lieber"`.

### Email body templates (Bluesky‑specific copy)

Two near‑identical templates — one for the table preview in verify mode, one actually sent — both hard‑code Bluesky URLs and Bluesky copy.

- `emailTemplatePreview()` — [main.go:237-253](cmd/send_campaign_emails/main.go): placeholder text `{BlueskyURL}` shown in `runVerify`.
- `generateEmailBody(r Recipient)` — [main.go:286-301](cmd/send_campaign_emails/main.go): interpolates `r.BlueskyURL` with `fmt.Sprintf`.

Key hard‑coded strings in the body:

- `"zuerichratsinfo ist jetzt auch auf Bluesky verfügbar:"`
- `"👉 https://bsky.app/profile/zuerichratsinfo.bsky.social"`
- `"... neu auch auf Bluesky, und markiert jeweils die Politiker..."`

### Send mode / SMTP

`runSend` (main.go:307-351) and helpers `buildMIMEMessage` / `sendEmail` (main.go:353-396) use Gmail SMTP over STARTTLS with `GMAIL_ADDRESS` + `GMAIL_APP_PASSWORD` env vars, with a `--delay`‑second pause between sends. Fully platform‑agnostic.

### Contact data model (multi‑platform ready)

[pkg/contacts/contacts.go:12-20](pkg/contacts/contacts.go)

```go
type Contact struct {
    Name      string
    X         []string
    Facebook  []string
    Instagram []string
    LinkedIn  []string
    Bluesky   []string
    TikTok    []string
}
```

- `GetPlatformURLs(name, platform string)` (contacts.go:101-124) already switches over `"x"/"twitter"`, `"facebook"`, `"instagram"`, `"linkedin"`, `"bluesky"`, `"tiktok"`.
- `HasPlatform(name, platform)` (contacts.go:127-129) returns `true` iff that platform has at least one URL for the contact.
- `GetAllContacts()` (contacts.go:132-134) returns every contact (used by the campaign mailer).

### Contact YAML conventions

[data/contacts.yaml](data/contacts.yaml) stores:

- `x:` as full X URLs (e.g. `https://x.com/AdinaRom`).
- `bluesky:` as full `https://bsky.app/profile/...` URLs.
- `instagram:` as full `https://www.instagram.com/<handle>/` URLs.
- `facebook:` as full `https://www.facebook.com/...` URLs (sometimes `profile.php?id=...`).
- `linkedin:` as full `https://www.linkedin.com/in/...` URLs.
- `tiktok:` as full URLs.

All platform lists are already `[]string`, so "first URL" selection (`contact.Instagram[0]`, etc.) is the established pattern — that's exactly what the Bluesky campaign does.

### Project social accounts referenced in README

[README.md:26-34](README.md)

| Platform    | Status  | Count | Project account                                              |
| ----------- | ------- | ----- | ------------------------------------------------------------ |
| LinkedIn    | Planned | 121   | —                                                            |
| Facebook    | Planned | 102   | —                                                            |
| Instagram   | Active  | 99    | `@zueriratsinfo` (`https://www.instagram.com/zueriratsinfo`) |
| X (Twitter) | Active  | 74    | `@zuerichratsinfo` (`https://x.com/zuerichratsinfo`)         |
| Bluesky     | Active  | 36    | `@zuerichratsinfo.bsky.social`                               |
| TikTok      | Planned | 21    | —                                                            |

Note the Instagram handle is `zueriratsinfo` (no `ch`), unlike the other active accounts.

### Verify/preview/send outputs

- `runVerify` (main.go:217-235) prints a `text/tabwriter` table with header columns `# / Name / Email / Gender / Salutation / Bluesky URL / Source`, then the template. The column header `"Bluesky URL"` is hard‑coded at main.go:220-221.
- `runPreview` (main.go:256-284) writes a Markdown document (stdout or `--output` file) with one `## N. Name` section per recipient followed by the rendered body.
- `runSend` logs `[i/N] ✅ Name <email>` per recipient and a final `Done: X sent, Y failed` line.

### README workflow (current, Bluesky‑specific)

[cmd/send_campaign_emails/README.md](cmd/send_campaign_emails/README.md) documents the 4‑step flow: verify → preview → test‑send to self → real send, including example outputs that reference `Bluesky URL` and `bsky.app/profile/...`.

## Code References

- [cmd/send_campaign_emails/main.go:29](cmd/send_campaign_emails/main.go) — hard‑coded subject `"zuerichratsinfo jetzt auch auf Bluesky"`.
- [cmd/send_campaign_emails/main.go:31-38](cmd/send_campaign_emails/main.go) — `Recipient` struct with `BlueskyURL`.
- [cmd/send_campaign_emails/main.go:76-83](cmd/send_campaign_emails/main.go) — filter `len(c.Bluesky) > 0`.
- [cmd/send_campaign_emails/main.go:105](cmd/send_campaign_emails/main.go) — `bskyURL := contact.Bluesky[0]`.
- [cmd/send_campaign_emails/main.go:183-213](cmd/send_campaign_emails/main.go) — name→email/gender lookup against PARIS API.
- [cmd/send_campaign_emails/main.go:237-253](cmd/send_campaign_emails/main.go) — Bluesky preview template.
- [cmd/send_campaign_emails/main.go:286-301](cmd/send_campaign_emails/main.go) — Bluesky send‑body template with `{BlueskyURL}` interpolation.
- [cmd/send_campaign_emails/main.go:353-396](cmd/send_campaign_emails/main.go) — generic SMTP send over Gmail STARTTLS.
- [pkg/contacts/contacts.go:12-20](pkg/contacts/contacts.go) — `Contact` with all six platform slices.
- [pkg/contacts/contacts.go:101-124](pkg/contacts/contacts.go) — `GetPlatformURLs` switch covering all six platforms.
- [data/email_overrides.yaml](data/email_overrides.yaml) — manual email/gender overrides, currently 6 entries, all added for the Bluesky run.
- [README.md:26-34](README.md) — project platform accounts and contact counts.

## Architecture Documentation

Pattern currently in use for the Bluesky campaign:

1. Load `data/contacts.yaml` via `contacts.LoadContacts`.
2. Filter `GetAllContacts()` to those with a non‑empty slice for the target platform.
3. Fetch the PARIS API contact list once (`zurichapi.Client.FetchAllKontakte`).
4. For each filtered contact: apply override (if any) else fuzzy‑match into the API list for email + `Geschlecht`.
5. Build `[]Recipient`, then dispatch to one of four modes (verify table / markdown preview / test‑send / real send).
6. Real send uses `smtp.Dial` + `StartTLS` + `PlainAuth` against `smtp.gmail.com:587`, with a configurable inter‑send delay.

Conventions:

- Platform URL lists in `contacts.yaml` are `[]string`; first element is treated as the canonical URL for tagging/linking.
- `Geschlecht == "weiblich"` → "Liebe", otherwise "Lieber" (no other cases handled).
- Overrides file is keyed by exact `name` from `contacts.yaml` (case‑insensitive via `strings.ToLower`).
- The campaign command writes only to stdout/stderr and (optionally) a `--preview --output` file; it does not persist any "already emailed" state.

## Historical Context (from thoughts/)

No prior research or plan documents in [thoughts/shared/](thoughts/shared/) reference `send_campaign_emails`, email campaigns, Instagram outreach, or the Bluesky mail run. Existing plans cover posting features (X reply threads, fraktion vote breakdown, dry post e2e, imagegen vertical spacing, visual vote posts) only.

## Open Questions

- Should the existing `cmd/send_campaign_emails` command be generalized (e.g. a `--platform` flag selecting Bluesky/Instagram/Facebook/TikTok/LinkedIn), or should each campaign be a separate `cmd/send_campaign_emails_<platform>/` binary?
  > generalizing the existing command is likely more efficient and less error‑prone than copy/pasting for each platform, but it would require refactoring the four hard‑coded Bluesky references (subject, filter, URL field, templates) into platform‑parameterized logic.
- Which handle/URL should the Instagram email reference — `@zueriratsinfo` / `https://www.instagram.com/zueriratsinfo` (current account per README.md) — and should the body still mention Bluesky and X as well, or only Instagram?
  > The current Instagram account is `@zueriratsinfo` (no `ch`), which is a bit inconsistent with the other platforms; if that's the official account, it should be used in the email copy and templates. The body could mention the other platforms as well.
- Should the mailer deduplicate against contacts that already received the Bluesky email (most Bluesky contacts also have Instagram), and if so, where should that "already emailed" state be persisted (new JSON in `data/`, a field in `contacts.yaml`, a per‑campaign log)?
  > no, they should not be deduplicated — each campaign is a separate announcement for a different platform, and it's likely that many contacts will want to receive both. The mailer currently does not persist any state, and adding that would add complexity; it's simpler to just run the Instagram campaign against all Instagram contacts regardless of Bluesky status.
- Should one email cover multiple new platforms at once (e.g. a single Instagram+future announcement) or should each platform launch get its own dedicated run?
  > not yet clear, for now the plan is to run separate campaigns for each platform, but the email body could be easily adapted to mention multiple platforms if desired. in the future I might want to use it to announce new features and then it would make sense to have a single campaign email covering all platforms.
- Is the current SMTP sending approach (personal Gmail + app password, 2s delay) acceptable for ~99 Instagram / ~102 Facebook / ~121 LinkedIn recipients, or does the larger volume warrant a different sender/throttling strategy?
  > for the current volumes, the Gmail approach with a 2s delay should be sufficient to avoid rate limits (which are typically around 100-150 emails per day for app passwords). If volumes grow significantly in the future, I might need to consider a more robust email sending service or implement batching with a longer delay.
- Should `data/email_overrides.yaml` be extended proactively for the Instagram run, or is the intended workflow to run verify mode first, see who's missing, and add overrides iteratively (as was done for Bluesky)?
  > the intended workflow is to run verify mode first, identify any missing or incorrect
