---
date: 2026-04-22
author: copilot
topic: "Generalize send_campaign_emails to target any social platform (Instagram first, Bluesky preserved)"
tags: [plan, send_campaign_emails, contacts, instagram, bluesky]
status: draft
research: thoughts/shared/research/2026-04-22-campaign-emails-per-platform.md
---

# Per-platform campaign emails — Implementation Plan

## Overview

Extend [cmd/send_campaign_emails/main.go](cmd/send_campaign_emails/main.go) so it can run a contact-announcement campaign for any supported social platform, not just Bluesky. First concrete use: an Instagram campaign for all ~99 contacts with an Instagram URL in `data/contacts.yaml`. Bluesky behavior must be preserved for parity/re-runs.

## Current State Analysis

The mailer is hard-coded to Bluesky in exactly four places:

- Subject constant: [main.go:29](cmd/send_campaign_emails/main.go) — `"zuerichratsinfo jetzt auch auf Bluesky"`.
- Contact filter: [main.go:76-83](cmd/send_campaign_emails/main.go) — `if len(c.Bluesky) > 0`.
- `Recipient.BlueskyURL` field + read-site: [main.go:31-38](cmd/send_campaign_emails/main.go), [main.go:105](cmd/send_campaign_emails/main.go).
- Body/template copy in `emailTemplatePreview()` and `generateEmailBody()`: [main.go:237-253](cmd/send_campaign_emails/main.go), [main.go:286-301](cmd/send_campaign_emails/main.go), plus the `"Bluesky URL"` column header in `runVerify` ([main.go:220-221](cmd/send_campaign_emails/main.go)).

Everything else (override YAML, PARIS API fuzzy match, gender→salutation, verify/preview/test/send modes, SMTP) is already platform-agnostic.

`pkg/contacts` is multi-platform-ready: [pkg/contacts/contacts.go:12-20](pkg/contacts/contacts.go) defines slices for all six platforms and [pkg/contacts/contacts.go:101-124](pkg/contacts/contacts.go) already switches over them in `GetPlatformURLs`.

## Desired End State

A single command invoked with `--platform {bluesky,instagram}` (extensible to facebook/tiktok/linkedin later):

```bash
go run cmd/send_campaign_emails/main.go --platform instagram
go run cmd/send_campaign_emails/main.go --platform instagram --preview --output data/emails_preview_instagram.md
go run cmd/send_campaign_emails/main.go --platform instagram --test you@gmail.com
go run cmd/send_campaign_emails/main.go --platform instagram --send
```

- Verify-mode table, preview Markdown, and sent body all reflect the chosen platform (subject, project handle URL, contact URL column, body copy).
- Running `--platform bluesky` produces **byte-identical** output to the pre-refactor command for the subject, verify template, and send body (modulo the now-parameterized column header, which becomes `"<Platform> URL"`).
- No persisted "already-emailed" state (per research answer — each campaign is independent).

### Key Discoveries

- Platform URL lists are already `[]string` and the convention is "first element is canonical" — directly reusable ([main.go:105](cmd/send_campaign_emails/main.go)).
- Instagram project handle per [README.md:26-34](README.md) is `@zueriratsinfo` (note: no `ch`), URL `https://www.instagram.com/zueriratsinfo`.
- The Bluesky body text mentions X as the prior/parallel platform; the Instagram body should analogously reference the already-live platforms (X and Bluesky) so recipients understand this is an expansion.
- No existing tests cover `cmd/send_campaign_emails`, so regression protection must come from the `--platform bluesky` golden-output comparison during manual verification.

## What We're NOT Doing

- No dedup against the previous Bluesky campaign run (per research answer).
- No Facebook / TikTok / LinkedIn platform configs yet — those are "Planned" in [README.md:26-34](README.md); adding them is a trivial follow-up once those accounts exist.
- No combined multi-platform announcement email (deferred per research answer).
- No change to SMTP sending, rate limits, or override file format.
- No new automated tests (package currently has none; adding harness is out of scope).

## Implementation Approach

Introduce a `platformConfig` struct and a package-level `platformConfigs` map keyed by the lowercase platform name. Each entry carries subject, display name (for column headers/logs), and two body-generating functions (or one function + a preview sentinel). The `Recipient.BlueskyURL` field becomes `PlatformURL`. `buildRecipientList` takes the selected config and uses `mapper.GetPlatformURLs(name, cfg.Key)` instead of reading `c.Bluesky` directly. Verify/preview/send all flow `cfg` through.

---

## Phase 1: Parameterize the command by platform (Bluesky-preserving refactor)

### Overview

Extract Bluesky-specific strings/logic into a `platformConfig` value, add a required `--platform` flag, and thread the config through all four modes without changing Bluesky output.

### Changes Required

#### 1. Add platform config type and registry

**File**: `cmd/send_campaign_emails/main.go`

```go
type platformConfig struct {
    Key         string // "bluesky", "instagram"
    DisplayName string // "Bluesky", "Instagram" — used in column header and logs
    Subject     string
    // Body takes the recipient's platform URL (first element of their list).
    // Returns the body *after* "{Salutation} {Name}\n\n".
    // When platformURL is empty (preview template), use a placeholder literal.
    Body func(platformURL string) string
}

var platformConfigs = map[string]platformConfig{
    "bluesky":   blueskyConfig,
    "instagram": instagramConfig, // added in Phase 2
}
```

Place `blueskyConfig` in a new file `cmd/send_campaign_emails/platform_bluesky.go` (keeps `main.go` focused on pipeline):

```go
var blueskyConfig = platformConfig{
    Key:         "bluesky",
    DisplayName: "Bluesky",
    Subject:     "zuerichratsinfo jetzt auch auf Bluesky",
    Body: func(url string) string {
        return fmt.Sprintf(`zuerichratsinfo ist jetzt auch auf Bluesky verfügbar:
👉 https://bsky.app/profile/zuerichratsinfo.bsky.social

Der Account publiziert die Abstimmungsresultate aus dem Gemeinderat auf X (https://x.com/zuerichratsinfo) und neu auch auf Bluesky, und markiert jeweils die Politikernnen und Politiker, welche die entsprechenden Vorstösse etc. eingereicht haben (wie dich: %s). Ziel ist es, politische Arbeit transparenter und für die Öffentlichkeit besser nachvollziehbar zu machen.

Wir arbeiten laufend daran, die Posts weiterzuentwickeln – zum Beispiel mit Statistiken, wie die einzelnen Fraktionen abgestimmt haben. Falls du Ideen oder Feedback hast, würde ich mich sehr darüber freuen!

Ich würde mich freuen, wenn du dem Account folgst. Und vielleicht hast du ja mal Lust einen Abstimmungspost mit deinen Followern zu teilen.

Weitere Informationen zum Projekt und eine Übersicht, wo alle GemeinderätInnen und StadträtInnen auf Social Media zu finden sind:
https://github.com/SiiiTschiii/zuerichratsinfo

Vielen Dank und liebe Grüsse
Christof
https://www.linkedin.com/in/christof-gerber/
`, url)
    },
}
```

Note: preserves the existing typo `Politikernnen` from [main.go:291](cmd/send_campaign_emails/main.go) so Bluesky output stays byte-identical for any future re-run. Fixing it is an intentionally-separate decision.

The verify-mode preview (where no concrete URL is bound) calls `blueskyConfig.Body("{BlueskyURL}")` — same placeholder string as today — and the verify-template's "Wir arbeiten laufend..." paragraph must still appear exactly as in [main.go:237-253](cmd/send_campaign_emails/main.go). Resolve this by having the verify path simply call `cfg.Body("{" + cfg.DisplayName + "URL}")` and accepting that the Bluesky verify template currently diverges slightly from the send template (see the trailing "Und vielleicht hast du..." sentence that only the send version has — a pre-existing inconsistency, not something to fix here).

**Behavior-preservation rule**: the Phase-1 refactor must produce the _send-mode_ body byte-identically to today for Bluesky. The _verify-mode_ preview may legitimately change to unify with the send body (it was already close-but-not-identical), since it's purely for operator inspection and already uses a placeholder. Document this in the PR description.

#### 2. Replace `Recipient.BlueskyURL` with `PlatformURL`

**File**: `cmd/send_campaign_emails/main.go` — [main.go:31-38](cmd/send_campaign_emails/main.go)

```go
type Recipient struct {
    Name        string
    Email       string
    Gender      string
    Salutation  string
    PlatformURL string
    Source      string
}
```

Update all read-sites ([main.go:105](cmd/send_campaign_emails/main.go), [main.go:223](cmd/send_campaign_emails/main.go), [main.go:301](cmd/send_campaign_emails/main.go)) accordingly.

#### 3. Add `--platform` flag and config lookup

**File**: `cmd/send_campaign_emails/main.go` — near [main.go:19-27](cmd/send_campaign_emails/main.go)

```go
platform = flag.String("platform", "", "Target platform (required): bluesky | instagram")
```

In `main()`, before dispatching:

```go
cfg, ok := platformConfigs[strings.ToLower(*platform)]
if !ok {
    log.Fatalf("--platform is required; supported: bluesky, instagram")
}
```

Thread `cfg` into `buildRecipientList`, `runVerify`, `runPreview`, `runSend`.

#### 4. Platform-agnostic contact filter

**File**: `cmd/send_campaign_emails/main.go` — replace [main.go:76-83](cmd/send_campaign_emails/main.go)

```go
var matching []contacts.Contact
for _, c := range allContacts {
    if len(mapper.GetPlatformURLs(c.Name, cfg.Key)) > 0 {
        matching = append(matching, c)
    }
}
fmt.Fprintf(os.Stderr, "Found %d contacts with %s accounts\n", len(matching), cfg.DisplayName)
```

And inside the loop ([main.go:105](cmd/send_campaign_emails/main.go)):

```go
urls := mapper.GetPlatformURLs(contact.Name, cfg.Key)
platformURL := urls[0]
```

Use `mapper.GetPlatformURLs` (already covers all six platforms — [pkg/contacts/contacts.go:101-124](pkg/contacts/contacts.go)) so no new switch is introduced here.

#### 5. Thread `cfg` through verify/preview/send

- `runVerify` — column header becomes `fmt.Sprintf("%s URL", cfg.DisplayName)` ([main.go:220-221](cmd/send_campaign_emails/main.go)), subject line prints `cfg.Subject`, template body is `cfg.Body("{" + cfg.DisplayName + "URL}")`.
- `runPreview` — title line `fmt.Fprintf(output, "# %s\n\n---\n\n", cfg.Subject)` ([main.go:272](cmd/send_campaign_emails/main.go)); body `cfg.Body(r.PlatformURL)`.
- `runSend` — subject arg `cfg.Subject`, body via `cfg.Body(r.PlatformURL)`.

Delete now-unused `emailTemplatePreview`, `generateEmailBody`, `generateFullEmail` and the `emailSubject` const.

### Success Criteria

#### Automated Verification

- [ ] `go build ./...` succeeds.
- [ ] `go vet ./...` is clean.
- [ ] `gofmt -l cmd/send_campaign_emails` produces no output.
- [ ] `go test ./...` passes (unchanged suite; only verifying nothing regressed in `pkg/contacts`).

#### Manual Verification

- [ ] `go run cmd/send_campaign_emails/main.go` (no `--platform`) exits non-zero with a usage error.
- [ ] `go run cmd/send_campaign_emails/main.go --platform bluesky` prints the same recipient count and table rows (same emails, genders, sources) as pre-refactor.
- [ ] `go run cmd/send_campaign_emails/main.go --platform bluesky --preview --output /tmp/bsky_new.md` output diffed against a pre-refactor snapshot shows **no** send-body changes (column header change and verify-template unification are expected and acceptable).
- [ ] `--platform bluesky --test <self>` delivers the same body as the previous Bluesky campaign email.

**Implementation Note**: Pause for manual verification before proceeding to Phase 2.

---

## Phase 2: Add Instagram platform config

### Overview

Register `instagramConfig` so `--platform instagram` selects all contacts with a non-empty `instagram:` list in `data/contacts.yaml` and sends an Instagram-announcement email.

### Changes Required

#### 1. Instagram config

**File**: `cmd/send_campaign_emails/platform_instagram.go` (new)

```go
package main

import "fmt"

var instagramConfig = platformConfig{
    Key:         "instagram",
    DisplayName: "Instagram",
    Subject:     "zuerichratsinfo jetzt auch auf Instagram",
    Body: func(url string) string {
        return fmt.Sprintf(`zuerichratsinfo ist jetzt auch auf Instagram verfügbar:
👉 https://www.instagram.com/zueriratsinfo

Der Account publiziert die Abstimmungsresultate aus dem Gemeinderat auf X (https://x.com/zuerichratsinfo), Bluesky (https://bsky.app/profile/zuerichratsinfo.bsky.social) und neu auch auf Instagram, und markiert jeweils die Politikerinnen und Politiker, welche die entsprechenden Vorstösse etc. eingereicht haben (wie dich: %s). Ziel ist es, politische Arbeit transparenter und für die Öffentlichkeit besser nachvollziehbar zu machen.

Wir arbeiten laufend daran, die Posts weiterzuentwickeln – zum Beispiel mit Statistiken, wie die einzelnen Fraktionen abgestimmt haben. Falls du Ideen oder Feedback hast, würde ich mich sehr darüber freuen!

Ich würde mich freuen, wenn du dem Account folgst. Und vielleicht hast du ja mal Lust einen Abstimmungspost mit deinen Followern zu teilen.

Weitere Informationen zum Projekt und eine Übersicht, wo alle GemeinderätInnen und StadträtInnen auf Social Media zu finden sind:
https://github.com/SiiiTschiii/zuerichratsinfo

Vielen Dank und liebe Grüsse
Christof
https://www.linkedin.com/in/christof-gerber/
`, url)
    },
}
```

Key copy choices (from research answers):

- Handle: `@zueriratsinfo` (no `ch`) — matches [README.md:26-34](README.md). This is intentional even though it differs from other project handles.
- Explicitly name **both** already-live platforms (X and Bluesky) so the email frames Instagram as an expansion rather than the first channel.
- Fix the `Politikernnen` typo in Instagram copy (`Politikerinnen`) since this is fresh text, not a regression of a shipped email.

#### 2. Register in map

**File**: `cmd/send_campaign_emails/main.go`

```go
var platformConfigs = map[string]platformConfig{
    "bluesky":   blueskyConfig,
    "instagram": instagramConfig,
}
```

### Success Criteria

#### Automated Verification

- [ ] `go build ./...` succeeds.
- [ ] `go vet ./...` is clean.
- [ ] `gofmt -l cmd/send_campaign_emails` produces no output.

#### Manual Verification

- [ ] `go run cmd/send_campaign_emails/main.go --platform instagram` lists ~99 recipients (matches [README.md:26-34](README.md) Instagram count, minus any without resolvable email).
- [ ] Verify table column reads `Instagram URL` and values look like `https://www.instagram.com/<handle>/`.
- [ ] `--platform instagram --preview --output data/emails_preview_instagram.md`: operator reads through, checks subject + project handle + URL interpolation for ~10 recipients.
- [ ] `--platform instagram --test <self>` delivers one mail per Instagram contact to self; subject reads "zuerichratsinfo jetzt auch auf Instagram", body links `@zueriratsinfo`, per-recipient URL is the contact's own Instagram profile.
- [ ] Operator iteratively adds any missing names to `data/email_overrides.yaml` (same workflow as Bluesky run, per research answer).
- [ ] Real send: `--platform instagram --send` — monitor `[i/N] ✅ ...` log; final line `Done: X sent, Y failed` with `Y == 0` or documented fails.

**Implementation Note**: Pause for manual verification (verify → preview → test-send → real send) before proceeding.

---

## Phase 3: Update command README

### Overview

Replace the Bluesky-only workflow doc at [cmd/send_campaign_emails/README.md](cmd/send_campaign_emails/README.md) with a platform-agnostic version that uses `--platform` throughout and lists supported platforms.

### Changes Required

#### 1. Rewrite README

**File**: `cmd/send_campaign_emails/README.md`

- Replace the one-line description with "Send personalized campaign emails to Gemeinderat members with an account on a given social platform, announcing zuerichratsinfo on that platform."
- Add a short "Supported platforms" list: `bluesky`, `instagram`. Mention Facebook / TikTok / LinkedIn as future additions gated on those accounts going live.
- Update every shell sample to include `--platform <name>` (use `instagram` as the running example, with a one-line note that `bluesky` also works).
- Update the example verify-table snippet's column header from `Bluesky URL` to `<Platform> URL`.
- Preserve the Overrides section unchanged — file format and matching semantics are identical.

### Success Criteria

#### Automated Verification

- [ ] `go run tools/check_md_tree.go` (if the repo uses it as a docs lint — verify before adding to the checklist) passes.
- [ ] Markdown renders correctly in the GitHub preview.

#### Manual Verification

- [ ] A fresh reader can follow the README for an Instagram campaign end-to-end.
- [ ] No residual Bluesky-only language outside the "supported platforms" list.

---

## Testing Strategy

### Unit Tests

None added. The command package has no existing test harness and the refactor is a pure string/plumbing change; investing in a mail-command test rig is disproportionate to the risk and out of scope.

### Integration Tests

None. The two integration surfaces (PARIS API fuzzy match and Gmail SMTP) are unchanged.

### Manual Testing Steps

1. **Phase 1 regression (Bluesky parity)**:
   - Before refactor, capture `go run cmd/send_campaign_emails/main.go > /tmp/bsky_old_verify.txt` and `--preview --output /tmp/bsky_old_preview.md`.
   - After refactor, run with `--platform bluesky` and diff. Only acceptable diffs: column header `"Bluesky URL"` unchanged, verify-template unified with send body (documented in PR).
2. **Phase 2 Instagram dry runs**:
   - Verify table (no env vars needed).
   - Preview to file, spot-check 10 recipients.
   - `--test <self>` and read 3–5 delivered mails.
3. **Phase 2 real send**: `--platform instagram --send`, monitor log output, cross-check `Done: X sent` against recipient count.

## References

- Research: [thoughts/shared/research/2026-04-22-campaign-emails-per-platform.md](thoughts/shared/research/2026-04-22-campaign-emails-per-platform.md)
- Campaign command: [cmd/send_campaign_emails/main.go](cmd/send_campaign_emails/main.go)
- Contacts mapper with multi-platform switch: [pkg/contacts/contacts.go:101-124](pkg/contacts/contacts.go)
- Project social accounts table: [README.md:26-34](README.md)
- Override file format: [data/email_overrides.yaml](data/email_overrides.yaml)
