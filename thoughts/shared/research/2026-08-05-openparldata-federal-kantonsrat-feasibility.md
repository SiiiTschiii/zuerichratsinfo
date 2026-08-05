---
date: 2026-08-05T12:00:00+02:00
researcher: claude
topic: "OpenParlData / Nationalrat / Kantonsrat ZH — feature-parity feasibility"
tags: [research, expansion, openparldata, kantonsrat, nationalrat, curiavista, feasibility]
status: complete — corrected against live APIs 2026-08-05
last_updated: 2026-08-05
follows_up: thoughts/shared/research/2026-07-06-expand-beyond-zurich.md
verified_by: thoughts/shared/research/2026-08-05-openparldata-api-verification.md
---

# Research: OpenParlData, Federal & Kantonsrat ZH — Feature-Parity Feasibility

**Date**: 2026-08-05
**Follows up on**: [2026-07-06-expand-beyond-zurich.md](2026-07-06-expand-beyond-zurich.md), open questions 1–3

> **⚠️ Corrected after live API verification.** This doc was originally written from source code alone, because the authoring sandbox blocked every relevant API host. All endpoints were later queried directly — see [2026-08-05-openparldata-api-verification.md](2026-08-05-openparldata-api-verification.md) for the full evidence. The routing recommendation below survived; several supporting facts did not. Corrections are marked inline as **✅ VERIFIED** / **❌ CORRECTED**. The headline changes:
>
> 1. **`MeaningYes`/`MeaningNo`/`Subject` return French in the German federal feed** (regression since Dec 2025) — a blocker for the federal post format.
> 2. **Kanton ZH is far more reliable than assumed** — 40/40 votings reconcile exactly at 180/180 members; the PDF/fuzzy-match fragility below is not visible in the live data.
> 3. **OpenParlData's Nationalrat data stops at 2025-09-26** (~10 months stale) — OData direct is the *only* working federal source, not merely the faster one.
> 4. Coverage is **22 cantons (not 24)** and **2 cities (not 1)** — Stadt Bern also has vote data.
> 5. **Sequencing flipped**: do Kanton ZH first, not the Nationalrat.

## Scope

Target jurisdictions: **Stadt Zürich (done)** → **Kanton Zürich (Kantonsrat)** → **Bund (Nationalrat)**.

**Feature-parity bar** (per project owner, narrower than the previous doc assumed): a new jurisdiction is viable if it supplies the *overall* vote result and the *per-Fraktion* breakdown. Per-politician vote records are **not** a requirement in themselves — they matter only as the raw material from which the Fraktion breakdown is computed (which is exactly how the Zurich city bot already works: `Stimmabgaben` → grouped by `Fraktion`).

## Method — and an important caveat

The sandbox's egress policy denies `ws.parlament.ch`, `api.openparldata.ch`, `opendata.swiss`, `parlzhcdws.cmicloud.ch` and `data.bs.ch` at the CONNECT level, for both `curl` and WebFetch. **No live API call was possible.** GitHub and GitLab *are* reachable, so this research instead went to the layer underneath the APIs:

- **OpenParlData's own ETL repository** — [`gitlab.com/opendata.ch/openparldatach/data-infrastructure`](https://gitlab.com/opendata.ch/openparldatach/data-infrastructure) (public, actively developed; last commit on the day of writing). It contains the FastAPI endpoint schemas, the Postgres body registry, and one Apache Hop pipeline per parliament. This is the *source of truth* for what the API can contain and where every field comes from.
- **The `parlament.ch` OData `$metadata`** and **OpenParlData's OpenAPI spec**, both vendored as test fixtures in [`metaodi/swissparlpy`](https://github.com/metaodi/swissparlpy).

Everything below about **schemas, field provenance and per-parliament coverage is verified from source code**. Everything about **row counts, freshness and data quality is not** — those need live queries. Concrete commands for that are in the last section.

> **✅ Update**: those live queries have since been run from an unrestricted network. Row counts, freshness and data quality are now measured, and the source-code-derived coverage claims turned out to be the *least* reliable part of this doc — the ETL repository describes pipelines that exist, not pipelines whose output actually reaches the API. See the verification doc.

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

> **❌ CORRECTED — this bonus does not survive contact with current data.** In OpenParlData it holds only for the Ständerat (XLSX-sourced, correct German) and for Nationalrat votes up to 2025-09-26, after which OPD stopped harvesting the NR entirely. Going to the OData source directly does **not** rescue it: since Dec 2025 `MeaningYes`/`MeaningNo` return **French** in the German feed (`"Adopter le projet"`). So for any vote a bot would post *today*, this field is unusable without a translation layer. Details and the monthly language breakdown are in §1 of the verification doc.

### Coverage: which parliaments actually have vote data

Derived by enumerating all 1,699 pipeline files under `hop/data/etl/stg/` and classifying by jurisdiction. The body registry (`database-seeding/snapshots/dwh_body.sql`, 2,411 rows) marks **78 bodies as `indexed`** (= "auto import enabled"): **50 cities + 26 cantons + 2 countries** (CHE, LIE) — matching the project's public "Bund + 26 Kantone + 50 Städte" claim.

Of those 78, the ones with vote pipelines:

| Level | Vote coverage (as claimed from pipeline files) |
| --- | --- |
| **Federal (CHE)** | ✅ votings + votes (NR via OData, SR via XLSX) |
| **Cantons** | ✅ **24 of 26** have vote and/or voting pipelines. Only **AI** and **NW** have none (both Landsgemeinde/small-parliament cantons). |
| **Cities** | ⚠️ **1 of 50 — Zürich (`261`) only.** |

That last row is the single most decision-relevant finding of this research: **among 50 indexed Swiss cities, Stadt Zürich is the only one whose roll-call votes OpenParlData ingests at all** — because Zurich's PARIS API is unusually good. The "other Swiss cities" branch of the original TODO is, as of today, essentially empty at the vote level regardless of which data route is chosen.

> **❌ CORRECTED.** `GET /v1/votings/group_by/body_key` returns **26 bodies with actual vote data**, not the pipeline-file count above:
>
> | Level | Verified coverage |
> | --- | --- |
> | **Countries** | CHE (26,015 votings) and **LIE** (4,376) — Liechtenstein is a second country, not a canton |
> | **Cantons** | **22 of 26.** Missing: AI, NW — **and also NE and VD**, which the pipeline files did not predict. Losing Neuenburg *and* Waadt matters for any later Romandie ambition. |
> | **Cities** | **2**: Zürich `261` (2,373) and **Bern `351` (1,941)** |
>
> Top cantons by volume: LU 6,336 · SG 4,356 · AG 3,465 · TI 3,349 · VS 3,182 · BL 2,891 · **ZH 2,626** · FR 2,119 · JU 1,958.
>
> **Stadt Bern is a real option this doc missed** — but it does not clear the feature-parity bar as-is: `person_parliamentary_group_name_de` is **null for all 80** members, so there is no Fraktion breakdown. `person_party_de` *is* fully populated (80/80) and could substitute. The broader claim still holds directionally: at the vote level, expansion to other Swiss cities is close to empty.

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

> **✅ VERIFIED — the diff was run and it passes.** PARIS reports `numHits=2370` with newest `SitzungDatum` 2026-07-08; OpenParlData `body_key=261` reports `total_records=2373` with newest `date` 2026-07-08. Title-joined comparison of the 40 newest from each source: **39 matched with identical `Schlussresultat`/`decision`, 0 conflicts**, 1 title outside the other's top-40 window (an ordering artifact — OPD sorts by a synthetic timestamp incrementing one minute per vote, PARIS by `seq`).
>
> So the adapter approach is trustworthy, *and* the "keep PARIS direct for the city" conclusion is confirmed: OPD is fully caught up here but adds nothing. Both feeds stop at 2026-07-08 because the Gemeinderat is in summer recess — that is not staleness.
>
> Practical gotchas found while calling PARIS by hand: `l=de-CH` is **mandatory** (`406 parameter language is mandatory` without it), the trailing slash on `searchdetails/` is **required** (otherwise a 301 to an internal `*.szh.loc` host), and the response is **XML**, not JSON.

### Verdict on Q1

**Use OpenParlData as the harmonized adapter for new jurisdictions, keep PARIS direct for Stadt Zürich.** One adapter unlocks the Kantonsrat and 23 other cantonal parliaments; the schema maps onto the existing domain model almost field-for-field. The reservations are real but manageable: beta status, self-declared "rough edges", unknown lag, and — for the specific jurisdictions this project cares about most after the city — a dependency on someone else's PDF scraper (see Q3).

Because the federal level has a first-class official API of its own, the sensible split is:

| Jurisdiction | Recommended source |
| --- | --- |
| Stadt Zürich (`261`) | **PARIS direct** (already built, fastest, canonical) |
| Bund / Nationalrat (`CHE`) | **parlament.ch OData direct** (see Q2) |
| Kanton Zürich (`ZH`) + any further canton | **OpenParlData** |

That is two new adapters, not one — but the second one (OpenParlData) then scales to 24 cantons at ~zero marginal cost.

> **✅ VERIFIED — this routing table is unchanged, and now better supported than the reasoning above manages.** Each row holds for a stronger reason: OPD is *provably* redundant for the city (39/40 identical), OData is the *only* working NR source (OPD's NR is 10 months stale), and Kanton ZH via OPD reconciles perfectly. Read "scales to 24 cantons" as **21 further cantons** (22 with data, minus ZH).

---

## Q2 — Federal Nationalrat

### Data path is confirmed and clean

`parlament.ch` OData at `https://ws.parlament.ch/odata.svc/` (no auth). From the vendored `$metadata`:

- **`Vote`** — one row per vote event: `ID`, `BusinessNumber`, `BusinessShortNumber`, `BusinessTitle`, `BusinessAuthor`, `BillTitle`, `Subject`, **`MeaningYes`**, **`MeaningNo`**, `VoteEnd`, `IdSession`, `IdLegislativePeriod`. **No aggregate counts.**
- **`Voting`** — one row per council member per vote: `PersonNumber`, `FirstName`, `LastName`, `Canton`, **`ParlGroupCode`**, **`ParlGroupName`**, **`ParlGroupNameAbbreviation`**, **`ParlGroupColour`**, `Decision`, `DecisionText`, plus the business fields denormalised.

So the **Ja/Nein totals and the Fraktion breakdown are both computed by aggregating `Voting` rows** — the identical pattern the bot already runs on `Stimmabgaben`. `Voting` ≈ `Stimmabgabe` almost field-for-field, and `ParlGroupColour` is a free win for `pkg/imagegen`.

> **✅ VERIFIED, with one field-level exception.** `Voting` returns exactly **200 rows** for vote 36558, so aggregate-from-members works as described. German is correct on `DecisionText` (`"Ja"`), `ParlGroupName` (`"FDP-Liberale Fraktion"`), `ParlGroupColour` (`"#FF00BFFF"`), `BusinessTitle` and `BusinessShortNumber`.
>
> **❌ But `Subject`, `MeaningYes` and `MeaningNo` return French** — see the correction under the classification note below. Those three fields are denormalised onto `Voting` rows too, so the problem follows you into either table.
>
> Also worth knowing before writing the client: the OData response shape is **inconsistent**. `$top`+`$orderby` returns `{"d": [...]}` while `$filter` returns `{"d": {"results": [...]}}`. A Go client must handle both.

OpenParlData does exactly this aggregation itself: `stg_load_curiavista_getvotes.hpl` contains an `Update totals voting` transform writing `voting_results_yes/no/absent/abstention` from the denormalised member votes, filtered to `voting_council_external_id = 'Council_1'` (= Nationalrat), with a sanity threshold of 195 (of 200 seats).

**Ständerat** is a separate path: `curiavista/votes_sr/` downloads **XLSX** files from parlament.ch, and *that* pipeline maps `results_yes/no/abstention/absent/total` directly. Consistent with SR name-lists only being published for a subset of votes. Treat SR as out of scope for v1.

### ~~Both routes work;~~ direct OData is better here — in fact it is the only one that works

| | OData direct | OpenParlData |
| --- | --- | --- |
| Freshness | during/right after session | ~~+ up to a daily harvest cycle~~ **❌ NR stops 2025-09-26** |
| Totals | compute yourself (same code as today) | pre-computed |
| `MeaningYes/No` | ~~✅~~ **❌ French since Dec 2025** | ✅ German, but only on data ≤ 2025-09 |
| Fraktion | `ParlGroupName` on each `Voting` row | `person_parliamentary_group_name_de` |
| Extra dep | none | beta third party |

Given the bot's value proposition is timeliness, and given the aggregation code already exists for Zurich, **use OData directly for the Nationalrat**.

> **✅ VERIFIED — right conclusion, but "both routes work" was wrong.** Only one route works.
>
> Federal votings in OPD use `external_id` = the bare numeric Curia Vista `Vote.ID` for the **Nationalrat** and `Council_2_<n>` for the **Ständerat**. Measured:
>
> - Newest **NR** voting in OPD: `35054`, **2025-09-26**
> - Newest **SR** voting in OPD: `Council_2_8366`, **2026-06-19** (current)
> - Newest NR vote in live OData: `Vote.ID` 36558, **2026-06-19**
>
> OpenParlData is missing roughly **10 months and ~1,500 Nationalrat votes**. Whatever broke the German text upstream appears to have broken OPD's NR harvest at the same time — the last good NR voting in OPD is dated the same week the language regression begins. So OData direct is not merely the fresher option for the NR, it is **the only working one**.

Practical note from `swissparlpy`'s README: the `Voting` table is large enough that unbounded queries return 500s; batch **per session** (`IdSession`) or per `IdVote`. The bot only ever needs a trailing window, so this is not a real constraint — but do not write a "fetch all" call.

### The editorial problem is the actual work

Federal sessions are 4 × 3 weeks a year. During a session the NR produces many votes per sitting day; between sessions, nothing. Posting every vote would flood every channel for three weeks and go silent for two months — and most NR votes are procedural (Ordnungsanträge, detail votes on individual articles) and unintelligible standalone.

Options, in preference order:

1. **Schlussabstimmungen only** — the final vote on a bill, typically the last day of a session. Low volume, high salience, self-explanatory, and `MeaningYes`/`MeaningNo` makes them postable without extra context. Recommended for v1.
2. **Schlussabstimmungen + Gesamtabstimmungen**, still a manageable volume.
3. **All votes with a rate limit + digest** — most faithful, most engineering, highest unsubscribe risk.

`Vote.Subject` and `Vote.MeaningYes/MeaningNo` are the fields to classify on; the exact marker for a Schlussabstimmung needs to be confirmed against live data (see verification section).

Volume figures are **not** verified here — the bot's posting cadence should be sized from a real query before committing to a policy.

> **❌ CORRECTED — classify on the French strings, and never on `Subject` alone.** On current data the marker is **`"Vote final"`** (and `"Vote sur l'ensemble"` for Gesamtabstimmungen), *not* `"Schlussabstimmung"`. A classifier matching the German word returns **zero rows** for everything since December 2025. Match both spellings, since the field was German up to 2025-09 and may revert if the upstream bug is fixed.
>
> `Subject` is also **null or empty in ~37%** of recent votes (111 null + 8 empty of 300 sampled), so it cannot carry the classification by itself.
>
> **✅ VERIFIED — volume, and the editorial concern is justified:**
>
> | Session | Total votes | Schluss-/Gesamtabstimmungen | Sitting days | Max in one day |
> | --- | ---: | ---: | ---: | ---: |
> | Frühjahrssession 2026 | 351 | 32 | 13 | **102** |
> | Wintersession 2025 | 313 | 42 | 13 | 56 |
> | Sommersession 2026 | 303 | 44 | 13 | 55 |
> | Sondersession 4. 2026 | 144 | 4 | 4 | 62 |
>
> ~300–350 votes per three-week session with peaks over 100/day confirms option 3 ("all votes") is not viable. **Option 1 (Schlussabstimmungen only) yields ~30–45 per session, ~150/year** — a workable cadence. The recommendation stands.

---

## Q3 — Kantonsrat Zürich

### Answered: the vote data exists ~~and it is PDF-derived~~ — and it is clean

The previous doc left this as the blocking unknown. It is now resolved — and the answer is *both* better and worse than hoped.

> **✅ CORRECTED — better than described below; the "worse" half did not materialise.** The pipeline analysis that follows is accurate about what exists in the ETL repo, but the live data does **not** show the fragility it predicts. Measured on the 40 most recent ZH votings:
>
> ```
> 40 of 40 reconcile EXACTLY (yes / no / abstention / absent all match the header totals)
> Every single voting returns exactly 180 member votes (= 180 seats)
> Unmapped Fraktion: 1–2 members per vote (~1%)
> ```
>
> Example — voting 104481 (Geschäftsbericht Regierungsrat 2025, the one genuinely contested vote in the sample): SVP 44/0/3 · SP 0/35/1 · FDP 27/0/3 · GLP 0/22/1 · Grüne 0/19/0 · Mitte 11/0/1 · EVP 0/6/1 · AL 0/5/0 · unmapped 1/0/0 → **83 Ja / 87 Nein / 0 Enth. / 10 abw., exactly matching `results_yes=83, results_no=87, results_abstention=0, results_absent=10`.**
>
> Moreover, `url_external_de` on every ZH voting points at **`https://zh.recapp.ch/shareparl?agendaItemUid=…&segmentUid=…`** — the Kantonsrat's structured recording/transcript system — not at an `AR*.pdf`. Together with GUID `external_id`s and flawless reconciliation, the ingest evidently no longer depends on the regex/fuzzy-match chain described below. (Inferred from the output; which pipeline actually runs upstream was not confirmed, and older data may still show PDF-era gaps — the sample is recent votes only.)
>
> **Net effect: Kanton ZH supports the full per-Fraktion post format, identical to the city bot.** Keep the completeness gate as cheap insurance, but do not build the product around a totals-only fallback being the common path.

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

> **Verified status of the three costs above:**
>
> - **Totals reliability** — no failures observed in 40/40. Risk unquantified but not currently materialising.
> - **Fuzzy-match gap** — **did not reproduce.** Keep the `sum(votes)` vs `sum(results_*)` check, but as insurance, not a load-bearing fallback.
> - **Missing fields** — **✅ confirmed.** `type_de`, `decision`, `meaning_of_yes/no` and `meeting_id` are all `null` for ZH. `decision` is trivially derived as stated. Vote type and "meaning of Yes" genuinely are unavailable.
>
> Two things this section gets wrong in the bot's favour:
>
> - **`affair_number` is reachable after all.** It is `null` *inline* on the voting, but `GET /v1/votings/{id}/affairs` returns the Geschäft with `number: "6087"`, `type_harmonized_de: "Regierungsgeschäft"`, `state_name_de: "Erledigt"` and a proper `kantonsrat.zh.ch` URL. Budget one extra request per vote.
> - **Field naming needs `lang_format=flat`.** Without it, member votes come back nested (`person_parliamentary_group_name: {de: "Fraktion SVP"}`) and every `*_de` lookup silently yields `null` — the exact trap the verification snippet at the end of this doc falls into. Always pass `lang_format=flat`.
>
> **New cost this section does not mention — harvest lag.** Measured over the last 100 ZH votings (`created_at` − `date`): **min 0.5 d, median 1.4 d, max 4.2 d.** The Kantonsrat sits Mondays, so votes typically surface Tuesday–Wednesday. The city bot posts within hours; Kantonsrat posts would run ~1.5 days behind. Not fatal, but it should shape the posting copy and the expectations set with users.

### Verdict on Q3

**Viable via OpenParlData, with a totals-first post format and a Fraktion-consistency check.** No need to contact Parlamentsdienste ZH as a prerequisite — though it remains worth doing as a durability play, since a first-party machine-readable endpoint would remove the PDF dependency for everyone. The direct-source alternative (parsing the AR PDFs in Go) is now clearly the *worse* option: same fragility, none of the maintenance shared.

> **✅ Upgraded verdict: viable via OpenParlData with the _full_ Fraktion post format**, not a totals-first one. Keep the consistency check. The "don't parse PDFs yourself" conclusion is now even more clear-cut, since the upstream ingest appears to have moved off PDFs entirely. The real trade-off for Kanton ZH is not data quality — it is the ~1.5 day lag.

---

## Feature-parity matrix

Fields the posting pipeline actually consumes today (measured by usage count across `pkg/voteposting` and `pkg/imagegen`) against what each new jurisdiction can supply:

| `zurichapi` field | uses | Stadt ZH (PARIS) | Kanton ZH (OPD) | Bund/NR (OData) |
| --- | ---: | --- | --- | --- |
| `Abstimmungstitel` | 24 | ✅ | ✅ `title_de` | ✅ `Subject` |
| `Schlussresultat` | 22 | ✅ | ⚠️ derive from counts | ⚠️ derive from counts |
| `Stimmabgaben` | 18 | ✅ | ✅ 180/180, exact <sup>†</sup> | ✅ `Voting` rows (200/200) |
| `AnzahlJa` / `AnzahlNein` | 26 | ✅ | ✅ `results_yes/no` | ✅ aggregate |
| `AnzahlEnthaltung` / `AnzahlAbwesend` | 22 | ✅ | ✅ `results_abstention/absent` | ✅ aggregate |
| `TraktandumTitel` | 11 | ✅ | ❌ | ❌ |
| `OBJGUID` (dedup) | 10 | ✅ | ✅ `external_id` | ✅ `Vote.ID` |
| `SitzungDatum` | 8 | ✅ | ✅ `date` | ✅ `VoteEnd` |
| `GeschaeftGrNr` | 7 | ✅ | ✅ `number` via `/affairs` <sup>‡</sup> | ✅ `BusinessShortNumber` |
| `GeschaeftTitel` | 6 | ✅ | ✅ `affair_title_de` | ✅ `BusinessTitle` |
| `GeschaeftGuid` | 3 | ✅ | ✅ `affair_id` | ✅ `BusinessNumber` |
| `Fraktion` | 3 | ✅ | ✅ `person_parliamentary_group_name_de` <sup>§</sup> | ✅ `ParlGroupName` |
| `Abstimmungsverhalten` | 1 | ✅ | ✅ `vote` | ✅ `DecisionText` |
| `Abstimmungstyp` | 1 | ✅ | ❌ | ⚠️ `MeaningYes/No` — **French only** |

**Both target jurisdictions clear the stated bar** (overall result + per-Fraktion). Losses are `TraktandumTitel` everywhere and `Abstimmungstyp` for the Kantonsrat; the federal level *gains* `MeaningYes/MeaningNo` and `ParlGroupColour`.

> <sup>†</sup> Verified exact on 40/40 recent votings, 180 members each — not merely "fuzzy-matched, hope for the best".
> <sup>‡</sup> Null inline; requires the extra `/v1/votings/{id}/affairs` call.
> <sup>§</sup> Requires `lang_format=flat`; 1–2 members per vote unmapped (~1%).
>
> **Corrected conclusion:** both jurisdictions still clear the bar, and **Kanton ZH clears it more comfortably than this table originally suggested**. But the federal "gain" is overstated — `MeaningYes/MeaningNo` is French-only on current data, so `Abstimmungstyp` is effectively a **loss** at both new levels until the upstream bug is fixed or a translation table is added. `ParlGroupColour` remains a genuine free win for `pkg/imagegen`.

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

> **❌ CORRECTED — swap the last two steps. Kanton ZH should ship before the Nationalrat.**
>
> The earlier doc called federal "the easiest, highest-quality expansion" and this one inherited that ordering. Verification reverses it. The Nationalrat now needs *three* things Kanton ZH does not:
>
> 1. a workaround for French `MeaningYes/MeaningNo`,
> 2. a vote-type classifier that copes with `Subject` being null ~37% of the time and French the rest,
> 3. an editorial policy for ~300 votes per session followed by two months of silence.
>
> Kanton ZH needs none of these — full Fraktion breakdowns, exact reconciliation, and the strongest audience synergy with the existing city product. The neutral-core refactor is identical either way, so nothing is wasted.
>
> **Revised sequencing:** extract the neutral core while still only serving Stadt Zürich → add the OpenParlData adapter, validating against PARIS on `261` (a known-good 39/40 diff) before pointing it at `ZH` → then tackle the Nationalrat once the French-text question is resolved.
>
> **Resolve the French text before committing to a federal post format.** Three options: report the bug to Parlamentsdienste and wait; ship a lookup table for the ~20 recurring `MeaningYes` phrases (`"Adopter le projet"` → `"Annahme des Entwurfs"`, `"Adopter la motion"` → `"Annahme der Motion"`, `"Proposition de la majorité"` → `"Antrag der Mehrheit"` — the vocabulary is small and highly repetitive; the top 6 phrases cover ~180 of 300 sampled votes); or drop the "what does Yes mean" line federally and lean on `BusinessTitle`, which is correctly German.

---

## Remaining verification (needs unrestricted network)

> **✅ ALL RESOLVED on 2026-08-05.** Every command below was run; results are in [2026-08-05-openparldata-api-verification.md](2026-08-05-openparldata-api-verification.md). Summary of the answers:
>
> | Question | Answer |
> | --- | --- |
> | OPD coverage | 26 bodies: CHE, LIE, **22 cantons**, **2 cities** (`261`, `351`) |
> | Kanton ZH freshness | median **1.4 d** behind the sitting (min 0.5, max 4.2) |
> | Kanton ZH completeness | **40/40 exact**, always 180/180 members; `len(votes) < 180` never observed |
> | Stadt Zürich diff vs PARIS | **39/40 identical**, 0 conflicts; OPD fully caught up |
> | Federal Schlussabstimmung marker | **`Subject == "Vote final"`** (French!), null ~37% of the time |
> | Federal session volume | 300–350 votes/session, peaks >100/day; ~30–45 Schluss-/Gesamtabstimmungen |
> | Are `results_*` ever NULL for NR? | **No** — never observed |
> | OPD rate limits | **No `RateLimit-*` or `Retry-After` headers exposed.** `server: uvicorn` behind a cache (`x-cache`). Several hundred sequential requests went through unthrottled; an hourly Action is very likely fine, but there is no published limit to rely on. **This one remains genuinely open** — worth asking the maintainers before depending on it. |
> | Auth | **None required** on any endpoint used |
>
> One correction to the snippets themselves: the Kanton ZH command below **omits `lang_format=flat`** on the `/votes` call, so `person_parliamentary_group_name_de` returns `null` for all 180 rows and the group-by looks empty. Fixed in the command list at the end of the verification doc.

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

**Also still open:** ~~OpenParlData rate limits and uptime expectations for an hourly GitHub Action; whether `results_*` is ever NULL for NR votings;~~ parlament.ch terms-of-use attribution wording for social posts.

Still genuinely open after verification:

- **OpenParlData rate limits / uptime SLA** — nothing published, no headers exposed. Ask the maintainers before depending on an hourly job.
- **parlament.ch terms-of-use attribution wording** for social posts (not a data question; needs a read of their terms).
- **The French-text regression** — report to Parlamentsdienste, and decide whether to wait for a fix or ship a translation table.
- **Whether older Kanton ZH data shows PDF-era gaps** — the 40/40 reconciliation covers recent votings only.

## Sources

Primary (read directly):
- [OpenParlData data-infrastructure (GitLab)](https://gitlab.com/opendata.ch/openparldatach/data-infrastructure) — `fastapi/app/schemas/endpoints/{votings,votes,bodies}.yaml`; `database-seeding/snapshots/dwh_body.sql`; `hop/data/etl/stg/CHE/{country/CHE_Schweiz/curiavista,canton/ZH_Zürich,city/261_Zürich}/`; `hop/data/etl/dwh/dwh_load_voting.hpl`
- [metaodi/swissparlpy](https://github.com/metaodi/swissparlpy) — `tests/fixtures/metadata.xml` (parlament.ch OData `$metadata`), `tests/fixtures/openapi.json` (OpenParlData OpenAPI spec), README

Secondary:
- [OpenParlData project page](https://opendata.ch/projects/openparldata/) · [API docs](https://api.openparldata.ch/documentation)
- [parlament.ch Open Data / Web Services](https://www.parlament.ch/de/über-das-parlament/fakten-und-zahlen/open-data-web-services) · [Abstimmungs-Datenbank NR](https://www.parlament.ch/de/ratsbetrieb/abstimmungen/abstimmungs-datenbank-nr)
- [zumbov2/swissparl](https://github.com/zumbov2/swissparl) · [Liip: Swiss Parliament Bot on OpenParlData](https://www.liip.ch/en/blog/new-api-new-scope-new-mcp-server-upgrading-the-swiss-parliament-bot)
