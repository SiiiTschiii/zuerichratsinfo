---
date: 2026-04-12T15:00:00+02:00
researcher: copilot
topic: "Per-Fraktion vote breakdown in X/Bluesky posts"
tags: [research, codebase, fraktion, voting, x, bluesky]
status: complete
last_updated: 2026-04-12
---

# Research: Per-Fraktion Vote Breakdown

**Date**: 2026-04-12

## Research Question

What data is available from the PARIS API for per-Fraktion vote breakdowns, and how could this be added to X/Bluesky posts?

## Summary

The PARIS API already returns individual council member votes (`Stimmabgaben`) nested inside each `Abstimmung` response. Each `Stimmabgabe` includes `Fraktion` and `Abstimmungsverhalten` fields. No additional API calls are needed — the data is already fetched but currently unused.

## API Data Available

### Stimmabgabe Fields (per council member)

From `pkg/zurichapi/types.go:119-130`:

| Field                  | Example Values                                                  |
| ---------------------- | --------------------------------------------------------------- |
| `Fraktion`             | `SP`, `FDP`, `GLP`, `Grüne`, `SVP`, `Die Mitte/EVP`, `AL`, `""` |
| `Partei`               | `SP`, `FDP`, `GLP`, `Grüne`, `SVP`, `Die Mitte`, `EVP`, `AL`    |
| `Abstimmungsverhalten` | `Ja`, `Nein`, `Enthaltung`, `Abwesend`                          |

### Real Example (2025/541 Schlussabstimmung, 79 Ja / 29 Nein)

```
Fraktion               Ja Nein Enth  Abw
-----------------------------------------------
SP                     32    0    0    5
Grüne                  17    0    0    1
GLP                    13    0    0    2
Die Mitte/EVP           8    0    0    2
AL                      8    0    0    0
(keine)                 1    0    0    0
SVP                     0   13    0    0
FDP                     0   16    0    7
```

125 Stimmabgaben total. 7 Fraktionen + 1 empty.

### Key Observations

- `Fraktion` groups parties into coalition factions (e.g. `Die Mitte/EVP` combines two parties)
- `Partei` is the individual party (more granular)
- Using `Fraktion` (not `Partei`) is better — fewer groups, politically relevant groupings
- Empty `Fraktion` exists (1 entry — likely Ratspräsident or special role)
- Data is already fetched via `FetchRecentAbstimmungen()` — no extra API call needed
- `Stimmabgaben` flows through `GroupAbstimmungenByGeschaeft()` to formatters untouched

## Post Format — Decided

### Format: Vertical List with Header Legend

One faction per line, header explains the number format. Includes all 4 values (Ja/Nein/Enthaltung/Abwesend).
Only for Ja/Nein votes (not Auswahl). Empty Fraktion omitted. Sorted by council seating order.
Full Fraktion names (no abbreviations — "Die Mitte/EVP" stays as-is).

### Thread Structure: Before and After

**BEFORE (current):**

```
[Root Post]
🗳️  Gemeinderat | Abstimmung vom 01.04.2026

✅ Angenommen: Liegenschaften Stadt Zürich, Schaffhauserstrasse 550, Instandsetzung Dächer

👇 Details im Thread

  [Reply 1]
  📊 79 Ja | 29 Nein | 0 Enthaltung | 17 Abwesend

  🔗 https://www.gemeinderat-zuerich.ch/abstimmungen/detail.php?aid=...
```

**AFTER (proposed):**

```
[Root Post]
🗳️  Gemeinderat | Abstimmung vom 01.04.2026

✅ Angenommen: Liegenschaften Stadt Zürich, Schaffhauserstrasse 550, Instandsetzung Dächer

👇 Details im Thread

  [Reply 1]
  📊 79 Ja | 29 Nein | 0 Enthaltung | 17 Abwesend

  🔗 https://www.gemeinderat-zuerich.ch/abstimmungen/detail.php?aid=...

  [Reply 2 — NEW]
  🏛️ Fraktionen (Ja/Nein/Enth/Abw):
  SP 32/0/0/5
  Grüne 17/0/0/1
  GLP 13/0/0/2
  Die Mitte/EVP 8/0/0/2
  AL 8/0/0/0
  SVP 0/13/0/0
  FDP 0/16/0/7
```

~175 chars — fits in 280 (X free) and 300 (Bluesky).

### Scenario: Single Vote (1 Abstimmung)

```
[Root Post]
🗳️  Gemeinderat | Abstimmung vom 01.04.2026

✅ Angenommen: Postulat von Reto Brüesch (SVP): Anpassung der Mindestfläche

👇 Details im Thread

  [Reply 1]
  📊 90 Ja | 30 Nein | 0 Enthaltung | 5 Abwesend

  🔗 https://www.gemeinderat-zuerich.ch/abstimmungen/detail.php?aid=...

  [Reply 2]
  🏛️ Fraktionen (Ja/Nein/Enth/Abw):
  SP 32/0/0/5
  Grüne 17/0/0/1
  GLP 13/0/0/2
  Die Mitte/EVP 8/0/0/2
  AL 8/0/0/0
  SVP 0/13/0/0
  FDP 0/16/0/7
```

### Scenario: Few Votes (2 Abstimmungen — multi-vote Geschäft)

Vote counts and Fraktion breakdown are two separate bin-packing entries.
If they fit in one reply, they stay together. If not, they split across two consecutive replies.

**When they fit together (~270 chars):**

```
[Root Post]
🗳️  Gemeinderat | Abstimmung vom 15.06.2025

Gesamtrevision der Gemeindeordnung

👇 Details im Thread

  [Reply 1]
  ✅ Einleitungsartikel
  📊 90 Ja | 20 Nein | 5 Enthaltung | 10 Abwesend

  🏛️ Fraktionen (Ja/Nein/Enth/Abw):
  SP 32/0/0/5
  Grüne 17/0/0/1
  GLP 13/0/0/2
  Die Mitte/EVP 8/0/0/2
  AL 8/0/0/0
  SVP 0/13/0/0
  FDP 0/16/0/7

  [Reply 2]
  ❌ Schlussabstimmung
  📊 40 Ja | 70 Nein | 5 Enthaltung | 10 Abwesend

  🏛️ Fraktionen (Ja/Nein/Enth/Abw):
  SP 10/22/0/5
  Grüne 5/13/0/0
  GLP 8/7/0/0
  Die Mitte/EVP 4/6/0/0
  AL 3/5/0/0
  SVP 0/13/0/0
  FDP 10/4/5/4

  🔗 https://www.gemeinderat-zuerich.ch/sitzungen/sitzung/?gid=...
```

**When they don't fit (long subtitle pushes vote counts too long):**

```
  [Reply 1]
  ✅ Einleitungsartikel
  📊 90 Ja | 20 Nein | 5 Enthaltung | 10 Abwesend

  [Reply 2]
  🏛️ Fraktionen (Ja/Nein/Enth/Abw):
  SP 32/0/0/5
  Grüne 17/0/0/1
  GLP 13/0/0/2
  Die Mitte/EVP 8/0/0/2
  AL 8/0/0/0
  SVP 0/13/0/0
  FDP 0/16/0/7

  [Reply 3]
  ❌ Schlussabstimmung
  📊 40 Ja | 70 Nein | 5 Enthaltung | 10 Abwesend

  [Reply 4]
  🏛️ Fraktionen (Ja/Nein/Enth/Abw):
  ...

  🔗 https://www.gemeinderat-zuerich.ch/sitzungen/sitzung/?gid=...
```

Link appended to the last reply.

### Scenario: Many Votes (10 Abstimmungen)

Same pattern — vote counts and Fraktion breakdown are separate bin-packing entries. The bin-packer tries to keep them together but splits if needed. With ~270 chars combined, most replies hold one vote+Fraktion pair. Link goes on the last reply.

### For Auswahl Votes (A/B/C type)

Same format, different header legend. `Abstimmungsverhalten` values are `A`, `B`, `C`, etc. instead of `Ja`/`Nein`.
Auswahl votes likely have only `Abwesend` (no `Enthaltung`) — current code shows `Abw.` only.
But we don't know for sure until we see real Stimmabgaben data for Auswahl votes.

```
🏛️ Fraktionen (A/B/C/Abw):
SP 20/10/2/5
Grüne 12/5/0/1
GLP 8/5/0/2
Die Mitte/EVP 4/4/0/2
AL 5/3/0/0
SVP 0/10/3/0
FDP 0/12/4/7
```

The aggregation logic is the same — group by `Fraktion`, count by `Abstimmungsverhalten`.
The header legend is built dynamically from the distinct `Abstimmungsverhalten` values present in the data.
No hardcoded column assumptions — columns adapt to whatever values the API returns.

## Implementation Approach

### Data Flow

1. `Abstimmung.Stimmabgaben.Stimmabgabe` already available in formatters
2. New function: `AggregateFraktionCounts([]Stimmabgabe) map[string]FraktionCounts`
3. New format function: `FormatFraktionBreakdown(map[string]FraktionCounts) string`
4. In `buildReplyPosts()`, vote counts and Fraktion breakdown are two separate bin-packing entries
5. Bin-packer tries to keep them in the same reply; if they don't fit, Fraktion breakdown goes in the next reply

### Where to Add Code

- `pkg/voteposting/voteformat/` — new `fraktion.go` with aggregation + formatting (shared between platforms)
- `pkg/voteposting/platforms/x/format.go` — call in `buildReplyPosts()`
- `pkg/voteposting/platforms/bluesky/format.go` — call in `buildReplyPosts()`
- New test fixtures with Stimmabgaben data in `testfixtures/fixtures.go`

### Fraktion Labels

Use full Fraktion names as returned by the API — no abbreviations.

| API Value       | Display         |
| --------------- | --------------- |
| `SP`            | `SP`            |
| `FDP`           | `FDP`           |
| `GLP`           | `GLP`           |
| `Grüne`         | `Grüne`         |
| `SVP`           | `SVP`           |
| `Die Mitte/EVP` | `Die Mitte/EVP` |
| `AL`            | `AL`            |
| `""`            | omit            |

## Code References

- `pkg/zurichapi/types.go:119-130` — Stimmabgabe struct with Fraktion field
- `pkg/zurichapi/types.go:88-116` — Abstimmung struct with nested Stimmabgaben
- `pkg/zurichapi/api.go:68-82` — FetchRecentAbstimmungen (includes Stimmabgaben)
- `pkg/zurichapi/api.go:166-248` — GroupAbstimmungenByGeschaeft (passes data through)
- `pkg/voteposting/platforms/x/format.go:127-214` — buildReplyPosts (insertion point)
- `pkg/voteposting/platforms/bluesky/format.go:134+` — Bluesky buildReplyPosts (insertion point)

## Decisions

- **Auswahl votes**: Supported — same format, header legend changes to `(A/B/C/Enth/Abw)` dynamically.
- **Empty Fraktion**: Omit — likely only Ratspräsident, not politically relevant.
- **Enthaltung/Abwesend**: Show all 4 values (Ja/Nein/Enth/Abw) — important for transparency.
- **Sort order**: By total members descending (biggest factions first, derived from data). Ties broken alphabetically. No hardcoded list.
- **Labels**: Full Fraktion names (no abbreviations — "Die Mitte/EVP" stays as-is).
- **Format**: Vertical list, one faction per line, with header legend `(Ja/Nein/Enth/Abw)`.
- **Thread placement**: New reply after vote counts + link reply.
