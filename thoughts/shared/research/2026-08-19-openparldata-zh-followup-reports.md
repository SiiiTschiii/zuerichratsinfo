---
date: 2026-08-19T22:00:00+02:00
researcher: claude
topic: "OpenParlData Kanton Zürich — follow-up reports after PR #56"
tags: [openparldata, kantonsrat, data-quality, upstream, bug-report]
status: drafts ready to file
follows_up: thoughts/shared/research/2026-08-08-openparldata-zh-data-gaps-report.md
upstream_tracker: https://gitlab.com/opendata.ch/openparldatach/data-infrastructure/-/issues
---

# Follow-up reports — what PR #56 turned up

Everything below was measured on **2026-08-19** against the live API and
`zh.recapp.ch`, over the 300 most recent `body_key=ZH` votings joined to the
archive on `url_external_de` → `agendaItemUid` → `extVotingUid`. All 300
resolved.

## Where each finding goes

| finding | issue | action |
| --- | --- | --- |
| `type_de` null on the 17.08.2026 sitting (new cause) | [#178](https://gitlab.com/opendata.ch/openparldatach/data-infrastructure/-/work_items/178) | comment |
| Attendance roll calls, annulled and empty votings | — | new issue |
| Ausgabenbremse indistinguishable from other `Quorum` votes | — | new issue |
| `/affairs` returns a bare `[]` | — | new issue |
| Cup-Abstimmung per-member records duplicated | [#180](https://gitlab.com/opendata.ch/openparldatach/data-infrastructure/-/work_items/180) | **already filed 2026-08-12** |
| `decision` still null; `votingResult` has four values | [#181](https://gitlab.com/opendata.ch/openparldatach/data-infrastructure/-/work_items/181) | comment |
| `Präsidium` maps to `abstention` | [#179](https://gitlab.com/opendata.ch/openparldatach/data-infrastructure/-/work_items/179) | comment, optional |

**Not filed:** PR #56 claimed `type_de` is "wrong in both directions". It does not
hold. Over the 300 votings `type_de` matches recapp's `votingScheme` in every
non-null case (Normal/binary 252, Quorum/quorum 31, Cup/cup 4). The
disagreement is between recapp's segment *title* and recapp's own
`votingScheme`, and the title is the loose one — 226 segments are titled only
"Abstimmung", including every vote on the preliminary support of an
Einzelinitiative. That was our bug, not theirs; fixed in `mergedType`.

---

## Comment on #178

> Still reproducing on 2026-08-19, but not from the pre-2023 population: 13 of the
> 300 most recent ZH votings have `type_de: null`, and every one of them is from
> November 2025 or later. Two causes, unrelated to each other.
>
> **1. `votingScheme` present, `type_de` null — 5 votings.** The whole sitting of
> 17.08.2026, ids 105409–105413. All five carry `votingScheme: "binary"` and
> `voting.voting_type: 0`, so the mapping in your table above should have reached
> them:
>
> ```bash
> curl -s "https://zh.recapp.ch/viewer/api/shareparl/segments?agendaItemUid=6e20a24f-3a9e-49ab-a855-269abd8457cd&ios=false&language=de" \
>   | jq -r '.[] | select(.extVotingUid != null) | [.extVotingUid, .title, .votingScheme] | @tsv'
> ```
>
> Their `updated_external_at` is 2026-08-18, so the rows were updated after the
> sitting and are still null today. That would fit `type_de` being written when the
> voting row is first inserted, with the protocol segments arriving later and
> nothing backfilling the type.
>
> **2. `votingScheme` absent, `voting_type: 5` — 8 votings.** ids 102615, 102617,
> 102708, 102709, 103981, 103982, 104473, 104477. No `votingScheme`, and `5` is not
> one of the codes the pre-2023 fallback maps. Five of them are attendance roll
> calls (`Anwesenheitsermittlung`, `Präsenzermittlung`, `Ermittlung der
> Anwesenden`), which are a problem of their own — filed separately as #NNN. The
> one that matters here is 102617, titled `Quorumsabstimmung` in the archive: the
> preliminary support of Einzelinitiative 137/2026 with 13 of 180 in favour, which
> without a type reads as a unanimous decision.
>
> **Where a type is served, your mapping holds.** Across those 300 votings `type_de`
> agrees with `votingScheme` in every non-null case — Normal/binary 252,
> Quorum/quorum 31, Cup-Abstimmung/cup 4, no exceptions.
>
> Two votings look mistyped at the source rather than by the import: 99904 (the
> preliminary support of Einzelinitiative 275/2025, 8 yes and 172 absent) and
> 102619 (segment title `Quorumsabstimmung`) both carry `votingScheme: "binary"`
> and are typed `Normal`. Where the segment `title` names a ballot type and
> disagrees with `votingScheme`, the title looks like the better of the two — but
> only then, since most segments are titled just "Abstimmung", including the
> threshold votes on preliminary support.
>
> ```bash
> curl -s "https://api.openparldata.ch/v1/votings/?body_key=ZH&limit=300&lang_format=flat" \
>   | jq -r '.data[] | select(.type_de==null) | [(.id|tostring), .date, .title_de] | @tsv'
> ```

---

## New issue A — non-decisions arrive as ordinary votings

> **Title:** Kanton Zürich (`body_key=ZH`): attendance roll calls, annulled and
> empty votings are published like decisions
>
> Following on from #178 — same importer, but a gap that fixing #178 will not
> close.
>
> Eight of the 300 most recent ZH votings are not decisions:
>
> | voting | recapp segment title | `type_de` | tally (Ja/Nein/Enth/Abw) |
> | --- | --- | --- | --- |
> | 104473, 104477 | `Anwesenheitsermittlung` | null | 174/0/0/6, 165/0/0/15 |
> | 103981, 103982 | `Präsenzermittlung` | null | 170/0/0/10, 163/0/0/17 |
> | 102615 | `Ermittlung der Anwesenden` | null | 160/0/0/20 |
> | 100464 | `Präsenzermittlung` | **Quorum** | 157/0/0/23 |
> | 101308 | `Präsenzabstimmung` | **Quorum** | 159/0/0/20 |
> | 101703 | `irrtümliche Quorumsabstimmung` | Quorum | 103/0/0/77 |
>
> The first five are covered by #178 only by accident: 100464 and 101308 show
> that a roll call gets typed `Quorum` once `votingScheme` is present, so after
> #178 all of them will be indistinguishable from a spending-brake vote. 101703
> is a vote the Kantonsrat itself marks as taken in error, which only its title
> says.
>
> Two further votings — 100972 and 101696 — report 0/0/0/180: every member
> absent, no vote recorded. recapp gives both `votingResult: "tie"`.
>
> Republished as they stand, a roll call becomes "170 Ja, 0 Nein, 10 Abwesend"
> under the headline of whatever business was on the agenda. We currently filter
> them by re-reading the archive's segment title ourselves.
>
> Anything that separates them would do — a `type_de` value carrying the
> archive's own label, or leaving them out of `/votings`.
>
> ```bash
> curl -s "https://api.openparldata.ch/v1/votings/104473?lang_format=flat" | jq '.data.type_de'
> curl -s "https://api.openparldata.ch/v1/votings/100464?lang_format=flat" | jq '.data.type_de'
> ```

---

## New issue B — the Ausgabenbremse is not identifiable

> **Title:** Kanton Zürich (`body_key=ZH`): the Ausgabenbremse cannot be told
> apart from other `Quorum` votes, and `meaning_of_yes_de` is null throughout
>
> `type_de: "Quorum"` covers two ballots that are counted differently and mean
> different things: the **Ausgabenbremse**, where a spending decision needs a
> fixed number of members in favour regardless of the Nein count, and the
> **preliminary support of an initiative**, where a much lower threshold
> applies. A reader cannot tell 128 in favour under a spending brake from 41 in
> favour of taking an Einzelinitiative further, though only one of them is a
> defeat.
>
> It is also not reliably `Quorum`. The two Ausgabenbremse votes of 17.08.2026
> arrive with `votingScheme: "binary"` in recapp, one of them 141 Ja to 1 Nein —
> so once #178 lands they will be typed `Normal` and become invisible:
>
> | voting | recapp title | `votingScheme` | `type_de` | tally |
> | --- | --- | --- | --- | --- |
> | 105409 | `Abstimmung Ausgabenbremse` | binary | null | 141/1/0/38 |
> | 105410 | `Abstimmung Ausgabenbremse` | binary | null | 172/0/0/8 |
> | 103926, 103928, 103930 | `Abstimmung Ausgabenbremse` | quorum | Quorum | 129/0/0/51, 128/0/0/52, 129/0/0/51 |
> | 101107 | `Ausgabenbremse` | quorum | Quorum | 174/0/0/6 |
>
> The archive names it in the segment `title`, which the ZH pipeline already
> reads. `meaning_of_yes_de` and `meaning_of_no_de` are null on all 300 ZH
> votings and would be the natural home for it if a new `type_de` value is too
> invasive.
>
> ```bash
> curl -s "https://api.openparldata.ch/v1/votings/?body_key=ZH&limit=300&lang_format=flat" \
>   | jq '[.data[] | select(.meaning_of_yes_de != null)] | length'
> # => 0
> ```

---

## New issue C — `/affairs` returns a bare array

> **Title:** `GET /v1/votings/{id}/affairs` returns a bare `[]` instead of the
> documented `{data, meta}` envelope
>
> A voting with an affair returns the documented envelope; a voting without one
> returns a bare empty array, so a client that decodes the documented shape
> fails on it:
>
> ```bash
> curl -s "https://api.openparldata.ch/v1/votings/105409/affairs?lang_format=flat" | jq 'keys'
> # => ["data","meta"]
> curl -s "https://api.openparldata.ch/v1/votings/105413/affairs?lang_format=flat"
> # => []
> ```
>
> It is not rare: 20 of the 300 most recent ZH votings have `affair_id: null`
> (procedural motions, Mitteilungen, roll calls). `{"data": [], "meta": {…}}`
> would make the empty case decodable like every other response.

---

## Comment on #180 — not needed

Already filed: SiiiTschiii's own comment of 2026-08-12 reports the duplicate
per-member records, with a cleaner measurement (175 `Präsidium` records per Cup
voting, counted by `person_id`). Nothing to add.

---

## Comment on #181 (optional)

> Still null for all of the 300 most recent ZH votings on 2026-08-19.
>
> Two notes from reading `votingResult` ourselves in the meantime:
>
> - It is not binary. Over the same 300 votings: `yes` ×244, `no` ×50, `tie` ×2
>   (100972, 101696 — both with no votes recorded at all) and `unknown` ×4 (the
>   four Cup-Abstimmung votings of #180).
> - For a quorum vote on an Einzelinitiative, `yes` means the initiative was
>   *vorläufig unterstützt* and goes to the Regierungsrat — not that it was
>   adopted. Mapping it onto a `decision` of "Ja"/"Angenommen" would overstate
>   it; we publish an outcome only for affair types where a carried vote really
>   is an adoption.

---

## Comment on #179 (optional)

> One small item for the harmonised vocabulary: ZH maps the chair's record
> (`vote_display_de: "Präsidium"`) to `abstention` rather than
> `abstention_president`. It occurs once per voting on 188 older ZH
> votings, so it inflates the abstention count by one on each.
>
> ```bash
> curl -s "https://api.openparldata.ch/v1/votes/?body_key=ZH&vote_display_de=Pr%C3%A4sidium&limit=1&lang_format=flat" \
>   | jq '.meta.total_records'
> ```
>
> (The bulk of those 683 records are the duplicates reported in #180.)
