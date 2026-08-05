---
date: 2026-08-05T12:00:00+02:00
researcher: claude
topic: "OpenParlData / Nationalrat / Kantonsrat ZH — feature-parity feasibility"
tags: [research, expansion, openparldata, kantonsrat, nationalrat, curiavista, feasibility]
status: complete
last_updated: 2026-08-05
follows_up: thoughts/shared/research/2026-07-06-expand-beyond-zurich.md
---

# Research: OpenParlData, Federal & Kantonsrat ZH — Feature-Parity Feasibility

**Date**: 2026-08-05
**Follows up on**: [2026-07-06-expand-beyond-zurich.md](2026-07-06-expand-beyond-zurich.md), open questions 1–3

## Scope

Target jurisdictions: **Stadt Zürich (done)** → **Kanton Zürich (Kantonsrat)** → **Bund (Nationalrat)**.

**Feature-parity bar** (per project owner, narrower than the previous doc assumed): a new jurisdiction is viable if it supplies the *overall* vote result and the *per-Fraktion* breakdown. Per-politician vote records are **not** a requirement in themselves — they matter only as the raw material from which the Fraktion breakdown is computed (which is exactly how the Zurich city bot already works: `Stimmabgaben` → grouped by `Fraktion`).

## Method — and an important caveat

The sandbox's egress policy denies `ws.parlament.ch`, `api.openparldata.ch`, `opendata.swiss`, `parlzhcdws.cmicloud.ch` and `data.bs.ch` at the CONNECT level, for both `curl` and WebFetch. **No live API call was possible.** GitHub and GitLab *are* reachable, so this research instead went to the layer underneath the APIs:

- **OpenParlData's own ETL repository** — [`gitlab.com/opendata.ch/openparldatach/data-infrastructure`](https://gitlab.com/opendata.ch/openparldatach/data-infrastructure) (public, actively developed; last commit on the day of writing). It contains the FastAPI endpoint schemas, the Postgres body registry, and one Apache Hop pipeline per parliament. This is the *source of truth* for what the API can contain and where every field comes from.
- **The `parlament.ch` OData `$metadata`** and **OpenParlData's OpenAPI spec**, both vendored as test fixtures in [`metaodi/swissparlpy`](https://github.com/metaodi/swissparlpy).

Everything below about **schemas, field provenance and per-parliament coverage is verified from source code**. Everything about **row counts, freshness and data quality is not** — those need live queries. Concrete commands for that are in the last section.

---

## Q1 — OpenParlData: harmonized adapter or per-source adapters?

### The `votings` schema is a near-exact match for what the bot needs

From `fastapi/app/schemas/endpoints/votings.yaml`, one row per vote event:

| Field | Note |
| --- | --- |
| `id`, `external_id`, `external_alternative_id` | stable IDs → dedup log |
| `body_key`, `body_id` | jurisdiction filter (`ZH`, `261`, `CHE`) |
| `date` | default sort field, `DESC` |
| `title_de/fr/it` | vote title |
| `type_de/fr/it` | vote type |
| **`results_yes`, `results_no`, `results_abstention`, `results_absent`** | **the overall result — the feature-parity requirement** |
| `results_string`, `decision` | textual result |
| `affair_id`, `affair_title_de/fr/it` | the Geschäft |
| **`meaning_of_yes_de/fr/it`, `meaning_of_no_de/fr/it`** | see "bonus" below |
| `url_external_de/fr/it` | link back to the official source |
| `meeting_id`, `group_id`, `updated_at`, `updated_external_at` | |

And `votes.yaml` (one row per member per vote) carries **`person_parliamentary_group_name_de`** (= Fraktion), `person_party_de`, `person_fullname` and `vote` (`yes`/`no`/`abstention`/`absent`) — so the **per-Fraktion breakdown is a group-by on `/v1/votings/{id}/votes`**, structurally identical to the current `GroupAbstimmungenByGeschaeft` → Fraktion logic.

Relevant endpoints: `/v1/votings/`, `/v1/votings/{id}/votes`, `/v1/votings/{id}/affairs`, `/v1/bodies/{id}/votings`, and `/v1/votings/group_by/body_key` (coverage counts in one call). Filtering is by arbitrary field (`?body_key=ZH&sort_by=-date&limit=50`), plus `fields=`, `expand=`, `output_format=csv|excel`. License **CC BY 4.0**, attribution `"Source: OpenParlData.ch"`. Beta, self-declared.

**Bonus that lands directly on an existing TODO**: `meaning_of_yes_*` / `meaning_of_no_*` is exactly the "explain what a Yes actually means" content the roadmap wants ("Explain the type of council business in the post"). It is populated for the federal level and Ständerat; **not** mapped for Kantonsrat ZH.

### Coverage: which parliaments actually have vote data

Derived by enumerating all 1,699 pipeline files under `hop/data/etl/stg/` and classifying by jurisdiction. The body registry (`database-seeding/snapshots/dwh_body.sql`, 2,411 rows) marks **78 bodies as `indexed`** (= "auto import enabled"): **50 cities + 26 cantons + 2 countries** (CHE, LIE) — matching the project's public "Bund + 26 Kantone + 50 Städte" claim.

Of those 78, the ones with vote pipelines:

| Level | Vote coverage |
| --- | --- |
| **Federal (CHE)** | ✅ votings + votes (NR via OData, SR via XLSX) |
| **Cantons** | ✅ **24 of 26** have vote and/or voting pipelines. Only **AI** and **NW** have none (both Landsgemeinde/small-parliament cantons). |
| **Cities** | ⚠️ **1 of 50 — Zürich (`261`) only.** |

That last row is the single most decision-relevant finding of this research: **among 50 indexed Swiss cities, Stadt Zürich is the only one whose roll-call votes OpenParlData ingests at all** — because Zurich's PARIS API is unusually good. The "other Swiss cities" branch of the original TODO is, as of today, essentially empty at the vote level regardless of which data route is chosen.

Note also that ~10 cantons derive votes from **PDF or XLSX scraping** (AG, FR, GE, GL, NE, OW, SO, SZ, TI, UR, ZG, **ZH**), which is a standing fragility for those jurisdictions.

### Cross-check against PARIS — a free validation path

`stg_load_city_insert_votes_261_Zürich.hpl` reads from:

```
https://www.gemeinderat-zuerich.ch/api/abstimmung/searchdetails/?q=seq%3E0
https://www.gemeinderat-zuerich.ch/abstimmungen/detail.php?aid=
```

That is **the exact endpoint `pkg/zurichapi` already calls** (`AbstimmungBaseURL`). So for Stadt Zürich, OpenParlData is a *re-serving* of PARIS, not an independent source. Consequences:

- Field-level equivalence for Stadt Zürich is structural, not a coincidence — a very cheap way to validate an OpenParlData adapter is to run it against `261` and diff the output against the existing PARIS client on the same votes.
- But it also means OpenParlData can only ever be *slower* than PARIS for Stadt Zürich, and adds a dependency without adding data. **Keep the direct PARIS client for Stadt Zürich.**

### Verdict on Q1

**Use OpenParlData as the harmonized adapter for new jurisdictions, keep PARIS direct for Stadt Zürich.** One adapter unlocks the Kantonsrat and 23 other cantonal parliaments; the schema maps onto the existing domain model almost field-for-field. The reservations are real but manageable: beta status, self-declared "rough edges", unknown lag, and — for the specific jurisdictions this project cares about most after the city — a dependency on someone else's PDF scraper (see Q3).

Because the federal level has a first-class official API of its own, the sensible split is:

| Jurisdiction | Recommended source |
| --- | --- |
| Stadt Zürich (`261`) | **PARIS direct** (already built, fastest, canonical) |
| Bund / Nationalrat (`CHE`) | **parlament.ch OData direct** (see Q2) |
| Kanton Zürich (`ZH`) + any further canton | **OpenParlData** |

That is two new adapters, not one — but the second one (OpenParlData) then scales to 24 cantons at ~zero marginal cost.

---

## Q2 — Federal Nationalrat

### Data path is confirmed and clean

`parlament.ch` OData at `https://ws.parlament.ch/odata.svc/` (no auth). From the vendored `$metadata`:

- **`Vote`** — one row per vote event: `ID`, `BusinessNumber`, `BusinessShortNumber`, `BusinessTitle`, `BusinessAuthor`, `BillTitle`, `Subject`, **`MeaningYes`**, **`MeaningNo`**, `VoteEnd`, `IdSession`, `IdLegislativePeriod`. **No aggregate counts.**
- **`Voting`** — one row per council member per vote: `PersonNumber`, `FirstName`, `LastName`, `Canton`, **`ParlGroupCode`**, **`ParlGroupName`**, **`ParlGroupNameAbbreviation`**, **`ParlGroupColour`**, `Decision`, `DecisionText`, plus the business fields denormalised.

So the **Ja/Nein totals and the Fraktion breakdown are both computed by aggregating `Voting` rows** — the identical pattern the bot already runs on `Stimmabgaben`. `Voting` ≈ `Stimmabgabe` almost field-for-field, and `ParlGroupColour` is a free win for `pkg/imagegen`.

OpenParlData does exactly this aggregation itself: `stg_load_curiavista_getvotes.hpl` contains an `Update totals voting` transform writing `voting_results_yes/no/absent/abstention` from the denormalised member votes, filtered to `voting_council_external_id = 'Council_1'` (= Nationalrat), with a sanity threshold of 195 (of 200 seats).

**Ständerat** is a separate path: `curiavista/votes_sr/` downloads **XLSX** files from parlament.ch, and *that* pipeline maps `results_yes/no/abstention/absent/total` directly. Consistent with SR name-lists only being published for a subset of votes. Treat SR as out of scope for v1.

### Both routes work; direct OData is better here

| | OData direct | OpenParlData |
| --- | --- | --- |
| Freshness | during/right after session | + up to a daily harvest cycle |
| Totals | compute yourself (same code as today) | pre-computed |
| `MeaningYes/No` | ✅ | ✅ |
| Fraktion | `ParlGroupName` on each `Voting` row | `person_parliamentary_group_name_de` |
| Extra dep | none | beta third party |

Given the bot's value proposition is timeliness, and given the aggregation code already exists for Zurich, **use OData directly for the Nationalrat**.

Practical note from `swissparlpy`'s README: the `Voting` table is large enough that unbounded queries return 500s; batch **per session** (`IdSession`) or per `IdVote`. The bot only ever needs a trailing window, so this is not a real constraint — but do not write a "fetch all" call.

### The editorial problem is the actual work

Federal sessions are 4 × 3 weeks a year. During a session the NR produces many votes per sitting day; between sessions, nothing. Posting every vote would flood every channel for three weeks and go silent for two months — and most NR votes are procedural (Ordnungsanträge, detail votes on individual articles) and unintelligible standalone.

Options, in preference order:

1. **Schlussabstimmungen only** — the final vote on a bill, typically the last day of a session. Low volume, high salience, self-explanatory, and `MeaningYes`/`MeaningNo` makes them postable without extra context. Recommended for v1.
2. **Schlussabstimmungen + Gesamtabstimmungen**, still a manageable volume.
3. **All votes with a rate limit + digest** — most faithful, most engineering, highest unsubscribe risk.

`Vote.Subject` and `Vote.MeaningYes/MeaningNo` are the fields to classify on; the exact marker for a Schlussabstimmung needs to be confirmed against live data (see verification section).

Volume figures are **not** verified here — the bot's posting cadence should be sized from a real query before committing to a policy.

---

## Q3 — Kantonsrat Zürich

### Answered: the vote data exists, and it is PDF-derived

The previous doc left this as the blocking unknown. It is now resolved — and the answer is *both* better and worse than hoped.

`hop/data/etl/stg/CHE/canton/ZH_Zürich/` contains `stg_load_canton_voting_ZH.hpl`, `stg_load_canton_insert_votes_ZH.hpl` and `stg_load_canton_votes_get_pdf_sub_ZH.hwf`. The chain is:

1. Select documents from the Kantonsrat's own document store where `doc_name LIKE 'AR%' OR doc_name LIKE 'Abstimmung%'` (AR = Abstimmungsresultat), excluding `VNL%`.
2. Download those PDFs and extract text.
3. Parse the header with a single regex:
   ```
   .*?häftstitel:(.*?)Geschäfts#:(.*?)Stimm-Datum[^\d]*?(\d{4}.\d{1,2}.\d{1,2}|\d{1,2}.\d{1,2}.\d{4})
   [^\d]*?(\d{1,2}.\d{1,2}.\d{2}).*?JA[^\d]*?(\d+).*?NE[^\d]*?(\d+)[^\d]*?(\d+)[^\d]*(\d+)[^\d]*(\d+).*?Stimme(.*)
   ```
   → Geschäftstitel, Geschäfts-Nr, Stimm-Datum, **JA count, NEIN count** and further counts.
4. Split the remaining text into per-voter lines, parse each with
   `^(.*?)\s(--|Ja|Nein|Enthaltung|Enthalten|enthalten|JA|NEIN|-|absent|ENTHALTEN)[^a-z].*`,
   then **fuzzy-match** the name against the Kantonsrat person table (a `FuzzyMatch` transform, with four name-format variants and an `authors_missing` reject file).

It writes `stg_voting` with `voting_results_yes`, `voting_results_no`, `voting_results_abstention`, `voting_results_absent`, `voting_affair_external_id`, `voting_affair_number`, `voting_affair_title_de`, `voting_date`, `voting_title_de`, `voting_url_de`, `voting_external_id` — and `stg_vote` with per-person `vote_vote` / `vote_person_external_id` / `vote_person_fullname`.

**So: feature parity for the Kantonsrat is achievable, and the PDF-parsing work the previous doc feared is already done and maintained upstream.** That removes the single biggest cost item from the Kantonsrat plan.

### What it costs

- **The overall result is as reliable as the PDF layout.** Totals come straight out of the header regex — robust as long as the Abstimmungsresultat template is stable, silently wrong or absent if it changes.
- **The Fraktion breakdown is only as good as the fuzzy name match.** Unmatched voters land in a reject file; the API would simply show fewer votes than seats. **A Fraktion breakdown that quietly drops members is worse than none** — the bot must validate `sum(votes) ≈ results_yes+no+abstention+absent` and fall back to totals-only when it doesn't. This is the single most important guardrail for a Kantonsrat integration.
- **Fields the Kantonsrat pipeline does *not* populate** (present for Stadt Zürich and the Bund): `decision`, `type_de`, `meaning_of_yes/no`, `meeting`/Traktandum. `decision` is trivially derivable (`yes > no`); the vote *type* (Schlussabstimmung/Ordnungsantrag/…) and the "what does Yes mean" text are **not available** for the Kantonsrat and would have to be dropped from the post format or sourced elsewhere.

### Verdict on Q3

**Viable via OpenParlData, with a totals-first post format and a Fraktion-consistency check.** No need to contact Parlamentsdienste ZH as a prerequisite — though it remains worth doing as a durability play, since a first-party machine-readable endpoint would remove the PDF dependency for everyone. The direct-source alternative (parsing the AR PDFs in Go) is now clearly the *worse* option: same fragility, none of the maintenance shared.

---

## Feature-parity matrix

Fields the posting pipeline actually consumes today (measured by usage count across `pkg/voteposting` and `pkg/imagegen`) against what each new jurisdiction can supply:

| `zurichapi` field | uses | Stadt ZH (PARIS) | Kanton ZH (OPD) | Bund/NR (OData) |
| --- | ---: | --- | --- | --- |
| `Abstimmungstitel` | 24 | ✅ | ✅ `title_de` | ✅ `Subject` |
| `Schlussresultat` | 22 | ✅ | ⚠️ derive from counts | ⚠️ derive from counts |
| `Stimmabgaben` | 18 | ✅ | ⚠️ fuzzy-matched | ✅ `Voting` rows |
| `AnzahlJa` / `AnzahlNein` | 26 | ✅ | ✅ `results_yes/no` | ✅ aggregate |
| `AnzahlEnthaltung` / `AnzahlAbwesend` | 22 | ✅ | ✅ `results_abstention/absent` | ✅ aggregate |
| `TraktandumTitel` | 11 | ✅ | ❌ | ❌ |
| `OBJGUID` (dedup) | 10 | ✅ | ✅ `external_id` | ✅ `Vote.ID` |
| `SitzungDatum` | 8 | ✅ | ✅ `date` | ✅ `VoteEnd` |
| `GeschaeftGrNr` | 7 | ✅ | ✅ `affair_number` | ✅ `BusinessShortNumber` |
| `GeschaeftTitel` | 6 | ✅ | ✅ `affair_title_de` | ✅ `BusinessTitle` |
| `GeschaeftGuid` | 3 | ✅ | ✅ `affair_id` | ✅ `BusinessNumber` |
| `Fraktion` | 3 | ✅ | ⚠️ via person join | ✅ `ParlGroupName` |
| `Abstimmungsverhalten` | 1 | ✅ | ✅ `vote` | ✅ `DecisionText` |
| `Abstimmungstyp` | 1 | ✅ | ❌ | ➕ `MeaningYes/No` instead |

**Both target jurisdictions clear the stated bar** (overall result + per-Fraktion). Losses are `TraktandumTitel` everywhere and `Abstimmungstyp` for the Kantonsrat; the federal level *gains* `MeaningYes/MeaningNo` and `ParlGroupColour`.

---

## What this implies for the refactor

The three sources agree closely enough that a small neutral core is obviously right, and — importantly — the shape is now known rather than guessed:

```
Vote        { SourceID, JurisdictionKey, Date, Title, Type,
              Yes, No, Abstention, Absent, Decision, SourceURL,
              Affair{ Number, Title, ID }, MemberVotes []MemberVote,
              MeaningYes, MeaningNo }      // optional, federal only
MemberVote  { Name, Party, Fraktion, Choice }
```

Points the research settles:

- **Totals must be first-class, not derived from `MemberVotes`.** Today they arrive together from PARIS; for the Kantonsrat they come from a *different parse step* than the member votes and can disagree. The formatters should read `Yes/No/…` directly and treat `MemberVotes` as optional enrichment.
- **The Fraktion breakdown needs a completeness gate.** `len(MemberVotes)` vs `Yes+No+Abstention+Absent`; below threshold, skip the breakdown reply rather than post a misleading one.
- **Optional fields are the norm, not the exception** — `Type`, `Meaning*`, Traktandum are absent for at least one jurisdiction each. Formatters must degrade, not branch per source.
- **Per-jurisdiction state**: `contacts.yaml`, `posted_votes_*.json` and platform credentials all need a jurisdiction dimension. `data/posted_votes_x.json` → `data/<jurisdiction>/posted_votes_x.json` keeps the existing dedup semantics intact.
- **Tagging does not port.** `contacts.yaml` is 132 curated Zurich city politicians. The Kantonsrat (180 seats) and the Nationalrat (200) each need their own curation effort — this remains the dominant *human* cost of expansion, unchanged by any of the above. OpenParlData's `persons` endpoint can seed names/party/Fraktion, but not social handles.

A concrete sequencing that keeps risk low: extract the neutral core **while still only serving Stadt Zürich** (pure refactor, existing tests as the safety net) → add the OData/Nationalrat adapter → add the OpenParlData adapter and validate it against PARIS on `261` before pointing it at `ZH`.

---

## Remaining verification (needs unrestricted network)

Everything below is a live-data question this sandbox could not answer. These are ready to paste.

**Coverage and volume, one call each:**
```bash
curl -s "https://api.openparldata.ch/v1/votings/group_by/body_key" | jq .
curl -s "https://api.openparldata.ch/v1/bodies/?indexed=true&fields=body_key,body_name_de,legislative_seats" | jq .
```

**Kanton ZH — freshness, completeness, and the fuzzy-match gap:**
```bash
curl -s "https://api.openparldata.ch/v1/votings/?body_key=ZH&sort_by=-date&limit=5&lang_format=flat" | jq .
# then, for the newest id N: do the member votes add up to the totals?
curl -s "https://api.openparldata.ch/v1/votings/N/votes?limit=1000" | jq '[.data[].person_parliamentary_group_name_de] | group_by(.) | map({g:.[0], n:length})'
```
Check: how far behind today is the newest vote (the PDF must be published first), and how often `len(votes)` < 180.

**Stadt Zürich — the free correctness check:**
```bash
curl -s "https://api.openparldata.ch/v1/votings/?body_key=261&sort_by=-date&limit=20&lang_format=flat" | jq .
```
Diff against `cmd/fetch_votes` output for the same votes: identical totals ⇒ the OpenParlData adapter is trustworthy; the date delta is the harvest lag to expect for `ZH`.

**Federal — vote-type classification and session volume:**
```bash
curl -s "https://ws.parlament.ch/odata.svc/Vote?\$top=20&\$orderby=VoteEnd%20desc&\$format=json&\$filter=Language%20eq%20'DE'" | jq .
```
Determine which field marks a Schlussabstimmung, and count votes per session day to size the editorial policy.

**Also still open:** OpenParlData rate limits and uptime expectations for an hourly GitHub Action; whether `results_*` is ever NULL for NR votings; parlament.ch terms-of-use attribution wording for social posts.

## Sources

Primary (read directly):
- [OpenParlData data-infrastructure (GitLab)](https://gitlab.com/opendata.ch/openparldatach/data-infrastructure) — `fastapi/app/schemas/endpoints/{votings,votes,bodies}.yaml`; `database-seeding/snapshots/dwh_body.sql`; `hop/data/etl/stg/CHE/{country/CHE_Schweiz/curiavista,canton/ZH_Zürich,city/261_Zürich}/`; `hop/data/etl/dwh/dwh_load_voting.hpl`
- [metaodi/swissparlpy](https://github.com/metaodi/swissparlpy) — `tests/fixtures/metadata.xml` (parlament.ch OData `$metadata`), `tests/fixtures/openapi.json` (OpenParlData OpenAPI spec), README

Secondary:
- [OpenParlData project page](https://opendata.ch/projects/openparldata/) · [API docs](https://api.openparldata.ch/documentation)
- [parlament.ch Open Data / Web Services](https://www.parlament.ch/de/über-das-parlament/fakten-und-zahlen/open-data-web-services) · [Abstimmungs-Datenbank NR](https://www.parlament.ch/de/ratsbetrieb/abstimmungen/abstimmungs-datenbank-nr)
- [zumbov2/swissparl](https://github.com/zumbov2/swissparl) · [Liip: Swiss Parliament Bot on OpenParlData](https://www.liip.ch/en/blog/new-api-new-scope-new-mcp-server-upgrading-the-swiss-parliament-bot)
