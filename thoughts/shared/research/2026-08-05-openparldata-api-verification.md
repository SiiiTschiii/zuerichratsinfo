---
date: 2026-08-05T18:00:00+02:00
researcher: claude
topic: "Live API verification of the OpenParlData / Nationalrat / Kantonsrat ZH feasibility plan"
tags: [research, verification, openparldata, kantonsrat, nationalrat, curiavista, paris]
status: complete
last_updated: 2026-08-05
verifies: thoughts/shared/research/2026-08-05-openparldata-federal-kantonsrat-feasibility.md
---

# Live API Verification

**Date**: 2026-08-05
**Verifies**: [2026-08-05-openparldata-federal-kantonsrat-feasibility.md](2026-08-05-openparldata-federal-kantonsrat-feasibility.md)

The feasibility doc was written from source code only — its sandbox blocked `api.openparldata.ch`, `ws.parlament.ch` and PARIS. All four hosts are reachable from here, so every open item in its "Remaining verification" section was run live. This doc records what the APIs actually return.

**Bottom line: the plan's recommendations survive, but three of its supporting facts are wrong and one new blocker appeared.** The recommended architecture (PARIS direct for the city, OData direct for the Nationalrat, OpenParlData for Kanton ZH) is still the right call — in two cases for stronger reasons than the plan gave.

---

## Summary of corrections

| # | Plan claim | Reality | Impact |
| --- | --- | --- | --- |
| 1 | `MeaningYes/No` + `Subject` give German "what a Yes means" text | **French-only since Dec 2025**, in the `Language='DE'` feed | **Blocker** for the federal post format |
| 2 | Kantonsrat ZH votes are PDF-scraped + fuzzy name-matched; needs a completeness guardrail | 40/40 votes reconcile **exactly**, always 180/180 members; source URL is `zh.recapp.ch`, not AR PDFs | Guardrail is cheap insurance, not a blocker — Kantonsrat is **easier** than planned |
| 3 | OpenParlData covers the federal level | NR data **stops 2025-09-26** (~10 months stale); only Ständerat is current | Strengthens "use OData direct for NR" |
| 4 | 24 of 26 cantons have vote data | **22 of 26**. Missing: AI, NW, **NE, VD** | Minor; VD/NE are notable absences |
| 5 | "1 of 50 cities — Zürich only" | **2 cities**: Zürich (2,373) and **Stadt Bern (1,941)** | New option the plan missed |

---

## 1. Federal `Subject` / `MeaningYes` / `MeaningNo` are French — a live regression

This is the one finding that changes what gets built.

`ws.parlament.ch/odata.svc` serves a per-language `Vote` entity. In the **German** feed (`Language eq 'DE'`), three fields now return French:

```
Vote 36558 (2026-06-19, Language='DE')
  Subject     = "Vote final"           ← expected "Schlussabstimmung"
  MeaningYes  = "Adopter le projet"    ← expected "Annahme des Entwurfs"
  MeaningNo   = "Rejeter le projet"
  BusinessTitle = "Abkommen zwischen dem Schweizerischen Bundesrat und dem
                   Ministerkabinett der Ukraine …"   ← correctly German
```

The `DE`, `FR` and `IT` variants return **byte-identical** `Subject`/`Meaning*` values (all French); only `BusinessTitle`/`BillTitle` are properly translated.

It is a regression, not a permanent design fact. Sampling the language of `Subject`+`MeaningYes` across the last 1,200 NR votes:

| Month | German | French | empty/other |
| --- | ---: | ---: | ---: |
| 2025-09 | **89** | 0 | 0 |
| 2025-12 | 9 | **282** | 22 |
| 2026-03 | 5 | **330** | 16 |
| 2026-04 | 0 | **138** | 6 |
| 2026-06 | 1 | **298** | 4 |

Spot checks confirm the older data is clean: vote 35054 (2025-09-26) → `Subject="Schlussabstimmung"`, `MeaningYes="Annahme der Vorlage"`; vote 30000 (2023-03-07) → `"Gesamtabstimmung"` / `"Annahme der Vorlage"`.

So the break lands between Herbstsession 2025 and Wintersession 2025 and is **still active in Sommersession 2026**.

**Consequences for the plan:**

- The plan's "bonus that lands directly on an existing TODO" — using `MeaningYes/MeaningNo` as the German explanation of what a Yes means — **does not work on current data**. A bot shipping today would post French into a German-language feed.
- The plan's classification advice ("`Vote.Subject` and `Vote.MeaningYes/MeaningNo` are the fields to classify on") needs the marker to be `"Vote final"` / `"Vote sur l'ensemble"`, **not** `"Schlussabstimmung"`. Any classifier must match both spellings or it will silently return zero rows for everything since December 2025.
- `Subject` is also **null or empty in ~37%** of recent votes (111 null + 8 empty of 300), so it cannot be the sole classifier.

Everything else on the federal side is fine and German: `DecisionText` (`"Ja"`), `ParlGroupName` (`"FDP-Liberale Fraktion"`), `ParlGroupColour` (`"#FF00BFFF"`), `BusinessTitle`, `BusinessShortNumber`. The per-member `Voting` table returns exactly **200 rows** for vote 36558, so the aggregate-from-members approach the plan describes works as stated.

Worth reporting to Parlamentsdienste — it is a plain data-quality bug affecting every consumer of the German feed.

## 2. Federal volume — the editorial concern is real, and now sized

The plan flagged the flood/silence problem but had no numbers. From `Vote` (Nationalrat, last 1,200 votes):

| Session | Total votes | Schluss-/Gesamtabstimmungen | Sitting days | Max in one day |
| --- | ---: | ---: | ---: | ---: |
| Frühjahrssession 2026 | 351 | 32 | 13 | **102** |
| Wintersession 2025 | 313 | 42 | 13 | 56 |
| Sommersession 2026 | 303 | 44 | 13 | 55 |
| Sondersession 4. 2026 | 144 | 4 | 4 | 62 |

~300–350 votes per three-week session with peaks over 100/day confirms "post everything" is not viable. The plan's recommended **Schlussabstimmungen-only** filter yields roughly **30–45 per session** (~150/year) — a workable cadence, and the recommendation stands.

## 3. Kanton Zürich is in much better shape than the plan assumed

The plan's Q3 is built on the Apache Hop PDF pipeline: regex-parsed AR PDFs plus a `FuzzyMatch` on member names, leading to its strongest warning — *"A Fraktion breakdown that quietly drops members is worse than none"* — and a mandatory completeness gate.

The live data does not behave that way.

**Reconciliation test — 40 most recent ZH votings, member votes vs. header totals:**

```
40 of 40 reconcile EXACTLY (yes/no/abstention/absent all match)
Every single voting returns exactly 180 member votes (= 180 seats)
Unmapped Fraktion: 1–2 members per vote (~1%)
```

Example (voting 104481, Geschäftsbericht Regierungsrat 2025 — the one genuinely contested vote in the sample):

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

Also, `url_external_de` on every ZH voting points at **`https://zh.recapp.ch/shareparl?agendaItemUid=…&segmentUid=…`** — the Kantonsrat's structured recording/transcript system — not at an `AR*.pdf`. Combined with GUID `external_id`s and flawless reconciliation, the ingest evidently no longer depends on the fragile PDF regex the plan describes.

**Revised guidance:** keep the completeness check (`len(votes)` vs `sum(results_*)`) as cheap insurance, but drop the assumption that a totals-only fallback will be the common path. On current evidence, Kanton ZH supports the **full** per-Fraktion post format, identical to the city bot.

**Caveats that do hold:**

- `type_de`, `decision`, `meaning_of_yes/no`, `meeting_id` are all `null` for ZH — the plan is right that vote type and "meaning of Yes" are unavailable. `decision` is trivially derived (`yes > no`).
- `affair_number` is `null` **inline**, but the plan's matrix is still effectively right: a second call to `/v1/votings/{id}/affairs` returns the Geschäft with `number: "6087"`, `type_harmonized_de: "Regierungsgeschäft"`, `state_name_de: "Erledigt"`, and a proper `kantonsrat.zh.ch` URL. Budget one extra request per vote.
- **Field naming**: member votes return nested objects (`person_parliamentary_group_name: {de: "Fraktion SVP"}`) unless you pass **`lang_format=flat`**, which flattens to `person_parliamentary_group_name_de`. The plan's verification snippet used the `_de` suffix *without* `lang_format=flat` and would have returned 180 nulls — as it did here on the first attempt. Always send `lang_format=flat`.

**Harvest lag** (`created_at` − `date`, last 100 ZH votings): **min 0.5 d, median 1.4 d, max 4.2 d.** The Kantonsrat sits Mondays; votes typically land in the API Tuesday–Wednesday. That is the real cost of the OpenParlData route — the city bot posts within hours, Kantonsrat posts would be ~1.5 days behind. Not fatal, but it should shape expectations and the posting copy.

## 4. OpenParlData's federal data is stale — NR stops in September 2025

The plan treats OpenParlData as a viable-but-slower federal alternative. It currently is not an alternative at all.

Federal votings use `external_id` = the bare numeric Curia Vista `Vote.ID` for the **Nationalrat**, and `Council_2_<n>` for the **Ständerat**.

- Newest **Nationalrat** voting: `35054`, **2025-09-26** — matching the last vote before the German→French regression.
- Newest **Ständerat** voting: `Council_2_8366`, **2026-06-19** — current.
- Live OData, meanwhile, has NR votes through **2026-06-19** (`Vote.ID` 36558).

So OpenParlData is missing roughly **10 months and ~1,500 Nationalrat votes**. Whatever broke the German text upstream appears to have broken OPD's NR harvest at the same time.

This makes the plan's federal recommendation stronger than it argued: **OData direct is not merely fresher for the Nationalrat, it is the only working source.**

(Nice detail: the NR votings OPD *does* have carry `meaning_of_yes_de: "Annahme der Vorlage"` in German — captured before the regression. OPD's Ständerat data, sourced from XLSX rather than OData, has correct German throughout.)

## 5. Coverage — two corrections

`GET /v1/votings/group_by/body_key` returns **26 bodies** with vote data:

- **1 country**: CHE (26,015) — plus LIE (4,376), which is a second country, not a canton.
- **22 cantons** — not the 24 the plan claimed. **Missing: AI, NW, NE, VD.** The plan predicted only AI and NW; **Neuenburg and Waadt** are also absent, which matters for any later Romandie ambition.
- **2 cities**: Zürich `261` (2,373) and **Bern `351` (1,941)**.

Top cantons by volume: LU 6,336 · SG 4,356 · AG 3,465 · TI 3,349 · VS 3,182 · BL 2,891 · **ZH 2,626** · FR 2,119 · JU 1,958.

**Stadt Bern is a genuine option the plan missed** — but it does not clear the feature-parity bar as-is:

```
Newest: 2026-07-02, "2025.SR.0271", yes=43 no=20 abstention=1 absent=15
80 member votes returned
person_party_de:                     80/80 populated  ("Grünes Bündnis", …)
person_parliamentary_group_name_de:   0/80 populated  (all null)
```

No Fraktion at all — though **party is fully populated** and could substitute. One caveat: member votes tally 2 abstentions against a header `results_abstention=1`, because a `"Präsidium"` vote is mapped to `abstention`. Small, but exactly the kind of off-by-one the plan's completeness gate is for. Harvest lag ~1.7 days.

## 6. Stadt Zürich — the free correctness check passes

The plan proposed diffing OpenParlData `261` against PARIS as a cheap trust check. Run:

- PARIS (`/api/abstimmung/searchdetails/`, the endpoint `pkg/zurichapi` already calls): `numHits=2370`, newest `SitzungDatum` **2026-07-08**.
- OpenParlData `body_key=261`: `total_records=2373`, newest `date` **2026-07-08T00:30:01**.
- Title-joined diff of the 40 newest from each: **39 matched with identical `Schlussresultat`/`decision`, 0 conflicts**, 1 title outside the other's top-40 window (an ordering artifact — OPD sorts by a synthetic timestamp that increments one minute per vote, PARIS by `seq`).

So OpenParlData faithfully re-serves PARIS and is fully caught up for the city. This validates the adapter approach — and equally confirms the plan's conclusion to **keep the direct PARIS client for Stadt Zürich**, since OPD adds a dependency without adding data.

Note both are at 2026-07-08 because Gemeinderat and Kantonsrat are in summer recess; the ZH figures above are recess-adjacent, not stale.

## 7. Odds and ends

- **Auth**: none required on either API. All calls above were unauthenticated.
- **Rate limits**: no `RateLimit-*` or `Retry-After` headers exposed. `server: uvicorn` behind a cache (`x-cache: MISS`, `x-cache-key`). Nothing blocked a few hundred sequential requests. An hourly GitHub Action is very likely fine, but there is **no published limit to rely on** — the plan's open question stands.
- **PARIS gotchas** (relevant if anyone hand-tests): `l=de-CH` is mandatory (`406 parameter language is mandatory` without it), the trailing slash on `searchdetails/` is required (otherwise a 301 to an internal `*.szh.loc` host), and the response is **XML**, not JSON.
- **OData response shape is inconsistent**: `$top`+`$orderby` returns `{"d": [...]}` while `$filter` returns `{"d": {"results": [...]}}`. A Go client must handle both.
- `results_*` was never NULL for any NR/SR voting sampled — one of the plan's open questions, resolved.

---

## Revised recommendation

The plan's routing table is unchanged and now better supported:

| Jurisdiction | Source | Status after verification |
| --- | --- | --- |
| Stadt Zürich (`261`) | PARIS direct | ✅ confirmed; OPD verified equivalent but redundant |
| Kanton ZH (`ZH`) | OpenParlData | ✅ **stronger than planned** — full Fraktion parity, ~1.5 d lag |
| Bund / NR (`CHE`) | parlament.ch OData | ✅ **only working option**; OPD's NR is 10 months stale |

Two changes to sequencing:

1. **Kanton Zürich should go first, not the Nationalrat.** The plan ordered federal first as "the easiest, highest-quality expansion". It is now the harder one: it needs a French-text workaround, a vote-type classifier that copes with `Subject` being null 37% of the time, and an editorial policy. Kanton ZH needs none of that — full Fraktion breakdowns, exact reconciliation, and the strongest audience synergy. The refactor is the same either way.
2. **Resolve the French-text question before committing to a federal post format.** Options: report the bug and wait; translate the ~20 recurring `MeaningYes` phrases with a lookup table (`"Adopter le projet"` → `"Annahme des Entwurfs"` etc. — the vocabulary is small and highly repetitive, see the frequency counts in §1); or drop the "what does Yes mean" line federally and rely on `BusinessTitle`, which is correctly German.

The domain-model guidance in the plan holds unchanged — totals first-class, optional fields the norm, per-jurisdiction dedup state. The completeness gate stays, but as insurance rather than a load-bearing fallback.

## Commands used

All reproducible without auth:

```bash
curl -s "https://api.openparldata.ch/v1/votings/group_by/body_key" | jq .
curl -s "https://api.openparldata.ch/v1/votings/?body_key=ZH&sort_by=-date&limit=40&lang_format=flat" | jq .
curl -s "https://api.openparldata.ch/v1/votings/104481/votes?limit=500&lang_format=flat" | jq .
curl -s "https://api.openparldata.ch/v1/votings/104481/affairs?lang_format=flat" | jq .
curl -s "https://ws.parlament.ch/odata.svc/Vote?\$top=300&\$orderby=VoteEnd%20desc&\$format=json&\$filter=Language%20eq%20'DE'" | jq .
curl -s "https://ws.parlament.ch/odata.svc/Voting/\$count?\$filter=IdVote%20eq%2036558%20and%20Language%20eq%20'DE'"
curl -s "https://www.gemeinderat-zuerich.ch/api/abstimmung/searchdetails/?q=seq%3E0%20sortBy%20seq/sort.descending&l=de-CH&s=1&m=40"
```
