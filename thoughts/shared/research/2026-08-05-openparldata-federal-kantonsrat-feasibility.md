---
date: 2026-08-05T12:00:00+02:00
researcher: claude
topic: "OpenParlData / Nationalrat / Kantonsrat ZH — feature-parity feasibility"
tags: [research, expansion, openparldata, kantonsrat, nationalrat, curiavista, feasibility]
status: complete
last_updated: 2026-08-05
follows_up: thoughts/shared/research/2026-07-06-expand-beyond-zurich.md
plan: thoughts/shared/plans/2026-08-05-source-neutral-votes-kanton-zurich.md
---

# Research: OpenParlData, Federal & Kantonsrat ZH — Feature-Parity Feasibility

**Date**: 2026-08-05
**Follows up on**: [2026-07-06-expand-beyond-zurich.md](2026-07-06-expand-beyond-zurich.md), open questions 1–3
**Plan**: [2026-08-05-source-neutral-votes-kanton-zurich.md](../plans/2026-08-05-source-neutral-votes-kanton-zurich.md)

## Scope

Target jurisdictions: **Stadt Zürich (done)** → **Kanton Zürich (Kantonsrat)** → **Bund (Nationalrat)**.

**Feature-parity bar** (per project owner, narrower than the previous doc assumed): a new jurisdiction is viable if it supplies the *overall* vote result and the *per-Fraktion* breakdown. Per-politician vote records are **not** a requirement in themselves — they matter only as the raw material from which the Fraktion breakdown is computed (which is exactly how the Zurich city bot already works: `Stimmabgaben` → grouped by `Fraktion`).

## Method

Two passes:

1. **Source-code analysis** of the layer underneath the APIs — OpenParlData's ETL repository ([`gitlab.com/opendata.ch/openparldatach/data-infrastructure`](https://gitlab.com/opendata.ch/openparldatach/data-infrastructure)), which holds the FastAPI endpoint schemas, the Postgres body registry, and one Apache Hop pipeline per parliament; plus the `parlament.ch` OData `$metadata` and OpenParlData's OpenAPI spec, both vendored as fixtures in [`metaodi/swissparlpy`](https://github.com/metaodi/swissparlpy).
2. **Live API verification** — every endpoint queried directly, unauthenticated, on 2026-08-05.

Pass 1 alone proved unreliable in a specific and instructive way: **the ETL repository describes pipelines that exist, not pipelines whose output actually reaches the API.** Three of its coverage conclusions were wrong (see §1.2, §2.3). Where the two passes disagree, this doc states the measured result and flags the discrepancy. Reproducible commands are in the last section.

---

## 1 — OpenParlData: harmonized adapter or per-source adapters?

### 1.1 The `votings` schema is a near-exact match for what the bot needs

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
| `meaning_of_yes_de/fr/it`, `meaning_of_no_de/fr/it` | see §2.2 — largely unusable in practice |
| `url_external_de/fr/it` | link back to the official source |
| `meeting_id`, `group_id`, `updated_at`, `updated_external_at` | |

`votes.yaml` (one row per member per vote) carries **`person_parliamentary_group_name_de`** (= Fraktion), `person_party_de`, `person_fullname` and `vote` (`yes`/`no`/`abstention`/`absent`) — so the **per-Fraktion breakdown is a group-by on `/v1/votings/{id}/votes`**, structurally identical to the current `GroupAbstimmungenByGeschaeft` → Fraktion logic.

Relevant endpoints: `/v1/votings/`, `/v1/votings/{id}/votes`, `/v1/votings/{id}/affairs`, `/v1/bodies/{id}/votings`, `/v1/votings/group_by/body_key`. Filtering is by arbitrary field (`?body_key=ZH&sort_by=-date&limit=50`), plus `fields=`, `expand=`, `output_format=csv|excel`. License **CC BY 4.0**, attribution `"Source: OpenParlData.ch"`. Beta, self-declared.

> ⚠️ **Always pass `lang_format=flat`.** Without it, multilingual fields come back nested (`person_parliamentary_group_name: {de: "Fraktion SVP"}`) and every `*_de` lookup silently yields `null` — an easy way to conclude "no Fraktion data" when it is right there.

### 1.2 Coverage: which parliaments actually have vote data

`GET /v1/votings/group_by/body_key` returns **26 bodies with vote data**:

| Level | Coverage |
| --- | --- |
| **Countries** | CHE (26,015 votings) and **LIE** (4,376) — Liechtenstein, not a canton |
| **Cantons** | **22 of 26.** Missing: **AI, NW, NE, VD** |
| **Cities** | **2**: Zürich `261` (2,373) and Bern `351` (1,941) |

By volume: LU 6,336 · SG 4,356 · AG 3,465 · TI 3,349 · VS 3,182 · BL 2,891 · **ZH 2,626** · FR 2,119 · JU 1,958 · BE 1,586 · BS 1,501.

Two notes against the pipeline-file analysis, which predicted 24 cantons and 1 city:

- **NE and VD are absent**, not just the expected Landsgemeinde cantons AI and NW. Losing Neuenburg *and* Waadt matters for any later Romandie ambition.
- **Stadt Bern has vote data.** It does not clear the parity bar as-is — `person_parliamentary_group_name_de` is **null for all 80 members** — but `person_party_de` is fully populated (80/80) and could substitute. One caveat found: member votes tally 2 abstentions against a header `results_abstention=1`, because a `"Präsidium"` vote maps to `abstention`.

The broader conclusion from the original TODO still holds directionally: **at the vote level, expansion to other Swiss cities is nearly empty.** Zurich and Bern are the whole set.

Note that ~10 cantons derive votes from PDF or XLSX scraping upstream (AG, FR, GE, GL, NE, OW, SO, SZ, TI, UR, ZG), a standing fragility for those jurisdictions.

### 1.3 Cross-check against PARIS — validated

`stg_load_city_insert_votes_261_Zürich.hpl` reads from `https://www.gemeinderat-zuerich.ch/api/abstimmung/searchdetails/?q=seq%3E0` — **the exact endpoint `pkg/zurichapi` already calls** (`AbstimmungBaseURL`). For Stadt Zürich, OpenParlData is a *re-serving* of PARIS, not an independent source.

The diff was run and passes:

- PARIS: `numHits=2370`, newest `SitzungDatum` **2026-07-08**.
- OpenParlData `body_key=261`: `total_records=2373`, newest `date` **2026-07-08**.
- Title-joined comparison of the 40 newest from each: **39 matched with identical `Schlussresultat`/`decision`, 0 conflicts.** The 1 non-match fell outside the other's top-40 window — an ordering artifact, since OPD sorts by a synthetic timestamp incrementing one minute per vote while PARIS sorts by `seq`.

Both feeds stop at 2026-07-08 because the Gemeinderat is in summer recess; that is not staleness.

**Consequences:**

- Field-level equivalence for Stadt Zürich is structural, not coincidental. This is a cheap, repeatable way to validate an OpenParlData adapter before pointing it at a jurisdiction we cannot cross-check.
- But OpenParlData can only ever be *slower* than PARIS for Stadt Zürich, and adds a dependency without adding data. **Keep the direct PARIS client for Stadt Zürich.**

> **PARIS gotchas** (relevant to anyone hand-testing): `l=de-CH` is **mandatory** (`406 parameter language is mandatory` without it); the trailing slash on `searchdetails/` is **required** (otherwise a 301 to an internal `*.szh.loc` host); the response is **XML**, not JSON.

### 1.4 Verdict

**Use OpenParlData as the harmonized adapter for new cantons, keep PARIS direct for Stadt Zürich, use OData direct for the Nationalrat.**

| Jurisdiction | Recommended source | Why |
| --- | --- | --- |
| Stadt Zürich (`261`) | **PARIS direct** | already built, fastest, canonical; OPD proven redundant (39/40 identical) |
| Kanton Zürich (`ZH`) | **OpenParlData** | full Fraktion parity, exact reconciliation (§3) |
| Bund / Nationalrat (`CHE`) | **parlament.ch OData direct** | OPD's NR data is ~10 months stale (§2.3) — OData is the *only* working source |

That is two new adapters, not one — but the OpenParlData one then scales to **21 further cantons** at ~zero marginal cost.

Reservations are real but manageable: beta status, self-declared "rough edges", and a harvest lag measured at ~1.5 days (§3.3).

---

## 2 — Federal Nationalrat

### 2.1 Data path

`parlament.ch` OData at `https://ws.parlament.ch/odata.svc/` (no auth):

- **`Vote`** — one row per vote event: `ID`, `BusinessNumber`, `BusinessShortNumber`, `BusinessTitle`, `BusinessAuthor`, `BillTitle`, `Subject`, `MeaningYes`, `MeaningNo`, `VoteEnd`, `IdSession`, `IdLegislativePeriod`. **No aggregate counts.**
- **`Voting`** — one row per council member per vote: `PersonNumber`, `FirstName`, `LastName`, `Canton`, `ParlGroupCode`, **`ParlGroupName`**, `ParlGroupNameAbbreviation`, **`ParlGroupColour`**, `Decision`, `DecisionText`, plus the business fields denormalised.

So **Ja/Nein totals and the Fraktion breakdown are both computed by aggregating `Voting` rows** — the identical pattern the bot already runs on `Stimmabgaben`. Verified: `Voting` returns exactly **200 rows** for vote 36558. `Voting` ≈ `Stimmabgabe` almost field-for-field, and `ParlGroupColour` is a free win for `pkg/imagegen`.

German is correct on `DecisionText` (`"Ja"`), `ParlGroupName` (`"FDP-Liberale Fraktion"`), `ParlGroupColour` (`"#FF00BFFF"`), `BusinessTitle` and `BusinessShortNumber`.

> **Implementation notes.** The OData response shape is inconsistent: `$top`+`$orderby` returns `{"d": [...]}` while `$filter` returns `{"d": {"results": [...]}}` — a Go client must handle both. Per `swissparlpy`'s README, unbounded `Voting` queries return 500s; batch per session (`IdSession`) or per `IdVote`. The bot only needs a trailing window, so this is not a real constraint — but do not write a "fetch all" call.

### 2.2 Blocker: `Subject` / `MeaningYes` / `MeaningNo` return French

In the **German** feed (`Language eq 'DE'`), three fields serve French:

```
Vote 36558 (2026-06-19, Language='DE')
  Subject       = "Vote final"            ← expected "Schlussabstimmung"
  MeaningYes    = "Adopter le projet"     ← expected "Annahme des Entwurfs"
  MeaningNo     = "Rejeter le projet"
  BusinessTitle = "Abkommen zwischen dem Schweizerischen Bundesrat und dem
                   Ministerkabinett der Ukraine …"    ← correctly German
```

The `DE`, `FR` and `IT` variants return **byte-identical** `Subject`/`Meaning*` values; only `BusinessTitle`/`BillTitle` are properly translated.

It is a regression, not a design fact. Language of `Subject`+`MeaningYes` across the last 1,200 NR votes:

| Month | German | French | empty/other |
| --- | ---: | ---: | ---: |
| 2025-09 | **89** | 0 | 0 |
| 2025-12 | 9 | **282** | 22 |
| 2026-03 | 5 | **330** | 16 |
| 2026-04 | 0 | **138** | 6 |
| 2026-06 | 1 | **298** | 4 |

Spot checks confirm older data is clean: vote 35054 (2025-09-26) → `"Schlussabstimmung"` / `"Annahme der Vorlage"`; vote 30000 (2023-03-07) → `"Gesamtabstimmung"` / `"Annahme der Vorlage"`. The break lands between Herbstsession 2025 and Wintersession 2025 and is **still active in Sommersession 2026**.

**Consequences:**

- The "explain what a Yes means" content the roadmap wants (`MeaningYes`/`MeaningNo`) **does not work on current data**. A bot shipping today would post French into a German-language feed.
- A Schlussabstimmung classifier must match **`"Vote final"`** (and `"Vote sur l'ensemble"`), *not* `"Schlussabstimmung"` — matching the German word returns **zero rows** for everything since December 2025. Match both, since the field may revert if the bug is fixed.
- `Subject` is **null or empty in ~37%** of recent votes (111 null + 8 empty of 300 sampled), so it cannot carry classification alone.

Worth reporting to Parlamentsdienste — a plain data-quality bug affecting every consumer of the German feed. Until then the options are: wait for a fix; a lookup table for the ~20 recurring `MeaningYes` phrases (the top 6 cover ~180 of 300 sampled votes, so this is small); or drop the line federally and lean on `BusinessTitle`.

### 2.3 OpenParlData is not a federal fallback

Federal votings in OPD use `external_id` = the bare numeric Curia Vista `Vote.ID` for the **Nationalrat**, and `Council_2_<n>` for the **Ständerat**.

- Newest **NR** voting in OPD: `35054`, **2025-09-26**
- Newest **SR** voting in OPD: `Council_2_8366`, **2026-06-19** (current)
- Newest NR vote in live OData: `Vote.ID` 36558, **2026-06-19**

OpenParlData is missing roughly **10 months and ~1,500 Nationalrat votes**. The last good NR voting in OPD is dated the same week the language regression begins, so whatever broke upstream likely broke both.

(The NR votings OPD *does* have carry `meaning_of_yes_de: "Annahme der Vorlage"` in correct German — captured before the regression. OPD's Ständerat data, sourced from XLSX rather than OData, has correct German throughout. **Ständerat remains out of scope for v1**: name lists are only published for a subset of votes by design.)

### 2.4 The editorial problem is the actual work

Federal sessions are 4 × 3 weeks a year. Measured volume:

| Session | Total votes | Schluss-/Gesamtabstimmungen | Sitting days | Max in one day |
| --- | ---: | ---: | ---: | ---: |
| Frühjahrssession 2026 | 351 | 32 | 13 | **102** |
| Wintersession 2025 | 313 | 42 | 13 | 56 |
| Sommersession 2026 | 303 | 44 | 13 | 55 |
| Sondersession 4. 2026 | 144 | 4 | 4 | 62 |

~300–350 votes per three-week session with peaks over 100/day, then two months of silence. Posting everything would flood every channel and most NR votes are procedural (Ordnungsanträge, article-level detail votes) and unintelligible standalone.

Options, in preference order:

1. **Schlussabstimmungen only** — ~30–45 per session (~150/year). Low volume, high salience, self-explanatory. **Recommended for v1.**
2. Schlussabstimmungen + Gesamtabstimmungen — still manageable.
3. All votes with a rate limit + digest — most faithful, most engineering, highest unsubscribe risk. Not viable at 102 votes/day.

---

## 3 — Kantonsrat Zürich

### 3.1 The data is complete and internally consistent

The upstream ETL repo contains a PDF-based chain for ZH (`stg_load_canton_votes_get_pdf_sub_ZH.hwf`): select documents matching `AR%`/`Abstimmung%`, extract text, parse the header with one large regex for JA/NEIN counts, split per-voter lines, then **fuzzy-match** names against the person table with a reject file for misses. That design predicts silently incomplete Fraktion breakdowns.

**The live data does not show that fragility.** Measured on the 40 most recent ZH votings:

```
40 of 40 reconcile EXACTLY (yes / no / abstention / absent all match header totals)
Every single voting returns exactly 180 member votes (= 180 seats)
Unmapped Fraktion: 1–2 members per vote (~1%)
```

Example — voting 104481 (Geschäftsbericht Regierungsrat 2025, the one genuinely contested vote in the sample):

| Fraktion | n | Ja | Nein | Abw. |
| --- | ---: | ---: | ---: | ---: |
| SVP | 47 | 44 | 0 | 3 |
| SP | 36 | 0 | 35 | 1 |
| FDP | 30 | 27 | 0 | 3 |
| Grünliberale | 23 | 0 | 22 | 1 |
| Grüne | 19 | 0 | 19 | 0 |
| Die Mitte | 12 | 11 | 0 | 1 |
| EVP | 7 | 0 | 6 | 1 |
| AL | 5 | 0 | 5 | 0 |
| *(unmapped)* | 1 | 1 | 0 | 0 |
| **Total** | **180** | **83** | **87** | **10** |

Header totals: `results_yes=83, results_no=87, results_abstention=0, results_absent=10`. **Exact match.**

Additionally, `url_external_de` on every ZH voting points at **`https://zh.recapp.ch/shareparl?agendaItemUid=…&segmentUid=…`** — the Kantonsrat's structured recording/transcript system — not at an `AR*.pdf`. Together with GUID `external_id`s and flawless reconciliation, the ingest evidently no longer depends on the regex/fuzzy chain.

> **Caveat on this inference**: which pipeline actually runs upstream was not confirmed, and the sample is recent votes only. Older data may still show PDF-era gaps. The completeness check below stays in regardless.

**Net: Kanton ZH supports the full per-Fraktion post format, identical to the city bot.**

### 3.2 What is missing

- **`type_de`, `decision`, `meaning_of_yes/no`, `meeting_id` are all `null`** for ZH. `decision` is trivially derived (`yes > no`). The vote *type* (Schlussabstimmung/Ordnungsantrag/…) and the "what does Yes mean" text genuinely are unavailable and must be dropped from the post format or sourced elsewhere.
- **`affair_number` is null inline**, but reachable: `GET /v1/votings/{id}/affairs` returns the Geschäft with `number: "6087"`, `type_harmonized_de: "Regierungsgeschäft"`, `state_name_de: "Erledigt"` and a proper `kantonsrat.zh.ch` URL. Budget one extra request per vote.
- **No Traktandum** equivalent (`TraktandumTitel` is used 11× in the current formatters).

### 3.3 Harvest lag is the real cost

Measured over the last 100 ZH votings (`created_at` − `date`): **min 0.5 d, median 1.4 d, max 4.2 d.**

The Kantonsrat sits Mondays, so votes typically surface Tuesday–Wednesday. The city bot posts within hours; Kantonsrat posts would run ~1.5 days behind. Not fatal, but it should shape the posting copy and the expectations set with users. This — not data quality — is the genuine trade-off for the Kantonsrat.

### 3.4 Verdict

**Viable via OpenParlData with the full Fraktion post format**, plus a completeness gate as cheap insurance. No need to contact Parlamentsdienste ZH as a prerequisite, though it remains worth doing as a durability play. The direct-source alternative (parsing AR PDFs in Go) is clearly worse: same fragility, none of the maintenance shared, and the upstream ingest appears to have moved off PDFs anyway.

---

## 4 — Feature-parity matrix

Fields the posting pipeline consumes today (usage count across `pkg/voteposting` and `pkg/imagegen`) against what each new jurisdiction supplies:

| `zurichapi` field | uses | Stadt ZH (PARIS) | Kanton ZH (OPD) | Bund/NR (OData) |
| --- | ---: | --- | --- | --- |
| `Abstimmungstitel` | 24 | ✅ | ✅ `title_de` | ✅ `Subject` (French) |
| `Schlussresultat` | 22 | ✅ | ⚠️ derive from counts | ⚠️ derive from counts |
| `Stimmabgaben` | 18 | ✅ | ✅ 180/180, exact | ✅ `Voting` rows (200/200) |
| `AnzahlJa` / `AnzahlNein` | 26 | ✅ | ✅ `results_yes/no` | ✅ aggregate |
| `AnzahlEnthaltung` / `AnzahlAbwesend` | 22 | ✅ | ✅ `results_abstention/absent` | ✅ aggregate |
| `TraktandumTitel` | 11 | ✅ | ❌ | ❌ |
| `OBJGUID` (dedup) | 10 | ✅ | ✅ `external_id` | ✅ `Vote.ID` |
| `SitzungDatum` | 8 | ✅ | ✅ `date` | ✅ `VoteEnd` |
| `GeschaeftGrNr` | 7 | ✅ | ✅ `number` via `/affairs` | ✅ `BusinessShortNumber` |
| `GeschaeftTitel` | 6 | ✅ | ✅ `affair_title_de` | ✅ `BusinessTitle` |
| `GeschaeftGuid` | 3 | ✅ | ✅ `affair_id` | ✅ `BusinessNumber` |
| `Fraktion` | 3 | ✅ | ✅ `person_parliamentary_group_name_de` | ✅ `ParlGroupName` |
| `Abstimmungsverhalten` | 1 | ✅ | ✅ `vote` | ✅ `DecisionText` |
| `Abstimmungstyp` | 1 | ✅ | ❌ | ⚠️ `MeaningYes/No` — French only |

**Both target jurisdictions clear the stated bar** (overall result + per-Fraktion), and Kanton ZH clears it comfortably. Losses: `TraktandumTitel` at both new levels, `Abstimmungstyp` effectively at both (unavailable for ZH, French-only federally). The federal level gains `ParlGroupColour`, a genuine free win for `pkg/imagegen`.

---

## 5 — What this implies for the refactor

The three sources agree closely enough that a small neutral core is obviously right, and the shape is now known rather than guessed:

```
Vote        { SourceID, JurisdictionKey, Date, Title, Type,
              Yes, No, Abstention, Absent, Decision, SourceURL,
              Affair{ Number, Title, ID }, MemberVotes []MemberVote,
              MeaningYes, MeaningNo }      // optional, federal only
MemberVote  { Name, Party, Fraktion, Choice }
```

This is the *eventual* shape across all three jurisdictions, not what to build first. `MeaningYes`/`MeaningNo` are federal-only and should be added with the Nationalrat adapter — the plan deliberately omits them, since no other source populates them and their federal content is currently French (§2.2), so even their type may change.

Points the research settles:

- **Totals must be first-class, not derived from `MemberVotes`.** Today they arrive together from PARIS; for the Kantonsrat they come from a different parse step than the member votes and *can* disagree in principle. Formatters should read `Yes/No/…` directly and treat `MemberVotes` as optional enrichment.
- **The Fraktion breakdown needs a completeness gate** — `len(MemberVotes)` vs `Yes+No+Abstention+Absent`; below threshold, skip the breakdown rather than post a misleading one. On current ZH data this never triggers, so it is insurance, not a load-bearing fallback. Stadt Bern's `Präsidium`→`abstention` off-by-one shows the failure mode is real elsewhere.
- **Optional fields are the norm, not the exception** — `Type`, `Meaning*`, Traktandum are absent for at least one jurisdiction each. Formatters must degrade, not branch per source.
- **Per-jurisdiction state**: `contacts.yaml`, `posted_votes_*.json` and platform credentials all need a jurisdiction dimension. `data/posted_votes_x.json` → `data/<jurisdiction>/posted_votes_x.json` keeps dedup semantics intact.
- **Tagging does not port.** `contacts.yaml` is 132 curated Zurich city politicians. The Kantonsrat (180 seats) and Nationalrat (200) each need their own curation effort — the dominant *human* cost of expansion, unchanged by any of the above. OpenParlData's `persons` endpoint can seed names/party/Fraktion, but not social handles.

**Sequencing: Kanton ZH before the Nationalrat.** The earlier doc called federal "the easiest, highest-quality expansion"; verification reverses that. The Nationalrat needs three things Kanton ZH does not — a French-text workaround, a classifier robust to a 37%-null `Subject`, and an editorial policy for 300 votes/session. Kanton ZH needs none, and has the strongest audience synergy with the existing product. The neutral-core refactor is identical either way, so nothing is wasted.

Concrete order: extract the neutral core **while still only serving Stadt Zürich** (pure refactor, existing tests as the safety net) → add the OpenParlData adapter, validating against PARIS on `261` before pointing it at `ZH` → then the Nationalrat, once the French-text question is resolved.

This is specified in [the implementation plan](../plans/2026-08-05-source-neutral-votes-kanton-zurich.md).

---

## 6 — Open questions

- **OpenParlData rate limits / uptime SLA.** Nothing published; no `RateLimit-*` or `Retry-After` headers exposed (`server: uvicorn` behind a cache). Several hundred sequential requests went through unthrottled, so an hourly Action is very likely fine — but there is no published limit to rely on. Ask the maintainers before depending on it.
- **parlament.ch terms-of-use attribution wording** for social posts.
- **The French-text regression** — report to Parlamentsdienste; decide whether to wait or ship a translation table.
- **Whether older Kanton ZH data shows PDF-era gaps** — the 40/40 reconciliation covers recent votings only.

Resolved by verification: OPD coverage; ZH freshness and completeness; the Stadt Zürich diff; the federal Schlussabstimmung marker; session volume; whether `results_*` is ever NULL for NR votings (**no**, never observed); auth requirements (**none** on any endpoint used).

---

## Sources

Primary — read directly:
- [OpenParlData data-infrastructure (GitLab)](https://gitlab.com/opendata.ch/openparldatach/data-infrastructure) — `fastapi/app/schemas/endpoints/{votings,votes,bodies}.yaml`; `database-seeding/snapshots/dwh_body.sql`; `hop/data/etl/stg/CHE/{country/CHE_Schweiz/curiavista,canton/ZH_Zürich,city/261_Zürich}/`; `hop/data/etl/dwh/dwh_load_voting.hpl`
- [metaodi/swissparlpy](https://github.com/metaodi/swissparlpy) — `tests/fixtures/metadata.xml` (parlament.ch OData `$metadata`), `tests/fixtures/openapi.json`, README

Secondary:
- [OpenParlData project page](https://opendata.ch/projects/openparldata/) · [API docs](https://api.openparldata.ch/documentation)
- [parlament.ch Open Data / Web Services](https://www.parlament.ch/de/über-das-parlament/fakten-und-zahlen/open-data-web-services) · [Abstimmungs-Datenbank NR](https://www.parlament.ch/de/ratsbetrieb/abstimmungen/abstimmungs-datenbank-nr)
- [zumbov2/swissparl](https://github.com/zumbov2/swissparl) · [Liip: Swiss Parliament Bot on OpenParlData](https://www.liip.ch/en/blog/new-api-new-scope-new-mcp-server-upgrading-the-swiss-parliament-bot)

## Verification commands

All reproducible without auth, as run on 2026-08-05:

```bash
# OpenParlData coverage
curl -s "https://api.openparldata.ch/v1/votings/group_by/body_key" | jq .

# Kanton ZH — freshness, completeness, Fraktion breakdown (note lang_format=flat)
curl -s "https://api.openparldata.ch/v1/votings/?body_key=ZH&sort_by=-date&limit=40&lang_format=flat" | jq .
curl -s "https://api.openparldata.ch/v1/votings/104481/votes?limit=500&lang_format=flat" \
  | jq '[.data[] | {f:.person_parliamentary_group_name_de, v:.vote}] | group_by(.f)
        | map({fraktion:.[0].f, total:length,
               yes:(map(select(.v=="yes"))|length), no:(map(select(.v=="no"))|length),
               abst:(map(select(.v=="abstention"))|length), abs:(map(select(.v=="absent"))|length)})'
curl -s "https://api.openparldata.ch/v1/votings/104481/affairs?lang_format=flat" | jq .

# Stadt Zürich — the free correctness check
curl -s "https://api.openparldata.ch/v1/votings/?body_key=261&sort_by=-date&limit=40&lang_format=flat" | jq .
curl -s "https://www.gemeinderat-zuerich.ch/api/abstimmung/searchdetails/?q=seq%3E0%20sortBy%20seq/sort.descending&l=de-CH&s=1&m=40"

# Federal — language regression, volume, per-member rows
curl -s "https://ws.parlament.ch/odata.svc/Vote?\$top=300&\$orderby=VoteEnd%20desc&\$format=json&\$filter=Language%20eq%20'DE'" | jq .
curl -s "https://ws.parlament.ch/odata.svc/Vote?\$filter=ID%20eq%2035054%20and%20Language%20eq%20'DE'&\$format=json" | jq .
curl -s "https://ws.parlament.ch/odata.svc/Voting/\$count?\$filter=IdVote%20eq%2036558%20and%20Language%20eq%20'DE'"
```
