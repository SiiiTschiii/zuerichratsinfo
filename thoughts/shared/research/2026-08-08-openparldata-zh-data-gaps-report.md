---
date: 2026-08-08T12:00:00+02:00
researcher: claude
topic: "OpenParlData Kanton Zürich importer — data gaps, verified and reported upstream"
tags: [openparldata, kantonsrat, data-quality, upstream, bug-report]
status: complete
last_updated: 2026-08-09
issue: https://github.com/SiiiTschiii/zuerichratsinfo/issues/46
follows_up: thoughts/shared/research/2026-08-05-openparldata-federal-kantonsrat-feasibility.md
upstream_tracker: https://gitlab.com/opendata.ch/openparldatach/data-infrastructure/-/issues
upstream_issues:
  - https://gitlab.com/opendata.ch/openparldatach/data-infrastructure/-/work_items/178
  - https://gitlab.com/opendata.ch/openparldatach/data-infrastructure/-/work_items/179
  - https://gitlab.com/opendata.ch/openparldatach/data-infrastructure/-/work_items/180
  - https://gitlab.com/opendata.ch/openparldatach/data-infrastructure/-/work_items/181
---

# OpenParlData Kanton Zürich — importer gaps

Companion to [issue #46](https://github.com/SiiiTschiii/zuerichratsinfo/issues/46). This
doc records the verification pass (all figures re-measured on **2026-08-08**), the
correction it forced to one of #46's assumptions, and the exact text filed upstream.

**Report A is fixed.** OpenParlData shipped `type_de` for ZH on 2026-08-09, within hours
of #178 being filed. Verified live: of the recent 200 ZH votings, `Normal` ×164,
`Quorum` ×24, `Cup-Abstimmung` ×4, null ×8. Two follow-ups came out of that fix — the
residual nulls (below) and report C, which #178's new `type_de` values made visible.

## Where the report goes

OpenParlData's "Found an error? Please report it here." link is not a contact form. It
opens a modal offering two routes into the same tracker — the **ETL repository's** issue
list, which is the repo the previous research pass already read the ZH pipelines from:

- `https://gitlab.com/opendata.ch/openparldatach/data-infrastructure/-/issues/new` (prefilled)
- `mailto:contact-project+opendata-ch-openparldatach-data-infrastructure-61320306-issue-@incoming.gitlab.com`

Filing needs a GitLab account, so it is a manual step.

| | report | upstream issue | status |
| --- | --- | --- | --- |
| A | `votings.type_de` is null for every ZH voting | [#178](https://gitlab.com/opendata.ch/openparldatach/data-infrastructure/-/work_items/178) | **fixed 2026-08-09** |
| B | `results_absent` conflates "nicht abgestimmt" with "nicht anwesend" | [#179](https://gitlab.com/opendata.ch/openparldatach/data-infrastructure/-/work_items/179) | open |
| C | Cup-Abstimmung has null aggregates; multi-option votes collapse under `vote` | [#180](https://gitlab.com/opendata.ch/openparldatach/data-infrastructure/-/work_items/180) | open |
| D | `decision` is null for every ZH voting, though recapp publishes `votingResult` | [#181](https://gitlab.com/opendata.ch/openparldatach/data-infrastructure/-/work_items/181) | open |

All four were filed on 2026-08-09. The "as filed" sections below are the text as
submitted; each was written before the next was filed, so their closing
cross-references name siblings without numbers.

### The #179 reply, and what it changes

Christian Gutknecht answered #179 on 2026-08-09 with the harmonised vocabulary
OpenParlData uses across 26 bodies:

| `vote` | meaning |
| --- | --- |
| `yes` / `no` / `abstention` | Ja / Nein / Enthaltung |
| `absent` | **keine Stimme abgegeben** — *no vote cast*, not *not present* |
| `further_option` | further option in a multiple choice |
| `absent_excused` | excused absence (CHE only) |
| `abstention_president` | the chair does not vote (CHE only) |

Two consequences for us.

**The documentation half of #179 is already answered.** `absent` means "cast no
vote", for any reason. That is exactly the fallback the report asked for — so
the field is not lying, *our label is*. We render it as "Abwesend", which asserts
physical absence, on every cantonal vote and not only quorum ones. Renaming that
column is ours to do, not upstream's.

**The proposed fix would change the shape of quorum votes.** He reads the
Kantonsrat's "Nicht abgestimmt" as the Nein button being pressed and recorded
under a nonsensical label, and proposes mapping it to `no` rather than `absent`.
If that lands, a quorum vote arrives as 128/46/6 instead of 128/0/52.

Our rendering is now written to survive either shape: `Nein` is folded into
"ohne Zustimmung" rather than read as a third column, so the 46 stay visible
whichever way upstream maps them. See `formatQuorumCounts` and `quorumChoice`.

**One doubt worth raising with him.** His reading is plausible but the evidence
cuts the other way: recapp's segments payload for voting 103928 has **no `n`
bucket at all** — 128 `y` and 52 `x`, nothing else. Had 46 members pressed Nein,
recapp would presumably record it. A simpler explanation is that the Kantonsrat
registers attendance separately — the `Anwesenheitsermittlung` and
`Präsenzermittlung` records we found while investigating the residual `type_de`
nulls do exactly that — so "Nicht anwesend" means *not registered present* and
"Nicht abgestimmt" means *registered present, no vote recorded*, with nobody
pressing anything. If so, mapping those 46 to `no` would assert they voted
against, which is a different false claim rather than a fix.

### Residual gap after #178

Eight of the recent 200 ZH votings are still `type_de: null`, and they are **not** the
pre-2023 PDF-era votings the fix accounted for — every one is 2026. All eight share a
single cause: recapp `voting_type: 5` with `votingScheme` absent entirely, so the
`votingScheme` mapping misses them and the pre-2023 `voting_type` fallback does not apply.

| voting | recapp segment title | tally |
| --- | --- | --- |
| 104473, 104477 | `Anwesenheitsermittlung` | 174/0/6, 165/0/15 |
| 103981, 103982 | `Präsenzermittlung` | 170/0/10, 163/0/17 |
| 102615 | `Ermittlung der Anwesenden` | 160/0/20 |
| 102708, 102709 | `Abstimmung` | 175/0/5, 170/0/10 |
| **102617** | **`Quorumsabstimmung`** | **13/0/167** |

This is the awkward residue: five are attendance determinations, which are not political
votes at all and which we publish as though they were, and `102617` is a real quorum vote
at 13 Ja / 0 Nein / 167 "absent" — the exact false-unanimity shape #46 was written about,
still unlabelled. Worth a comment on #178 rather than a new issue.

**Two issues, not one.** #46 suggested reporting them together because both come from
the ZH importer. Verification argued the other way: the two defects draw on different
sources (recapp's JSON vs. the official PDF), sit in different pipeline files, and are
very different in size — gap 1 looks like propagating a value already staged, gap 2
raises a schema question about harmonised vote values. Bundled, neither could be closed
until both were. Each report below is self-contained and mentions the other in passing.

## What changed against #46

#46 proposed that both gaps could be closed from recapp's `segments` endpoint, since it
returns `votingScheme` joined on `extVotingUid`. That holds for gap 1 and **not** for
gap 2.

recapp's per-member results are three codes — `y` / `n` / `x`. For the 11:42 quorum vote
it reports `x ×52`, the same collapsed figure OpenParlData publishes. The 6-vs-46 split
survives only in the official PDF. So gap 2 is not a join OpenParlData is failing to
make; it is data that only the PDF carries — and the ZH pipeline already downloads those
PDFs. The report below says so, rather than asking for a fix that would not work.

## Verification

Everything below was re-measured, not carried over from #46.

| claim | verified |
| --- | --- |
| `type_de` null for ZH | 200/200 sampled votings, none populated |
| `type_de` populated for Stadt Zürich | 200/200: `Normal` ×187, `Quorum` ×11, 2 × multi-option |
| voting 103928 reports 52 absent | `results_absent: 52`, and 52 member records `vote: "absent"` / `Abwesend` |
| official PDF splits them | header reads `Nicht anwesend 6`, `Abgestimmt 128` |
| 104490 is an ordinary 167:0 | confirmed, `type_de` null, 13 absent |
| recapp exposes the vote type | `votingScheme` `binary`/`quorum`; `voting.voting_type` `0`/`4`; segment title `Abstimmung` / `Abstimmung Ausgabenbremse` |
| recapp does **not** expose the absent split | per-member codes are `y`/`n`/`x` only |

The chamber is 180 seats and the `/votes` endpoint returns 180 records, so the PDF's
`Nicht abgestimmt` count is `180 − 6 − 128 = 46`. That number is derived, and the report
labels it as derived.

### Pipeline reading

From `gitlab.com/opendata.ch/openparldatach/data-infrastructure`, branch `develop`
(note: **not** `main` — the API 404s on that ref):

- `hop/data/etl/stg/CHE/canton/ZH_Zürich/stg_load_canton_insert_protocols_segements_ZH.hpl`
  calls `https://zh.recapp.ch/viewer/api/shareparl/segments?agendaItemUid=…` and extracts
  `$.*.extVotingUid`, `$.*.title`, `$.*.type` and `$.*.voting`. It does **not** extract
  `$.*.votingScheme` — but `$.*.title` and `$.*.voting` (which contains `voting_type`)
  are already staged, so the type is arguably in the warehouse already and simply never
  reaches `votings.type_de`.
- `stg_load_canton_insert_votes_ZH.hpl` maps `voting_results_absent → total_absent`, and
  its only per-member text pattern is
  `^(.*?)\s(--|Ja|Nein|Enthaltung|Enthalten|enthalten|JA|NEIN|-|absent|ENTHALTEN)[^a-z].*`
  — no branch for `Quorum`, `Nicht abgestimmt` or `Nicht anwesend`.
- `stg_load_canton_extract_pdf.hpl` and `stg_load_canton_votes_get_pdf_sub_ZH.hwf` show
  the PDF path already exists.
- By contrast `stg_load_city_insert_votes_261_Zürich.hpl` carries a `voting_type_de`
  field. The canton pipeline has no equivalent, which is what makes this look like an
  omission rather than a decision.

## Reproduction

```bash
# 1. type_de is null for ZH, populated for Stadt Zürich
curl -s "https://api.openparldata.ch/v1/votings/?body_key=ZH&limit=200&lang_format=flat" \
  | jq '[.data[].type_de] | group_by(.) | map({v: .[0], n: length})'
curl -s "https://api.openparldata.ch/v1/votings/?body_key=261&limit=200&lang_format=flat" \
  | jq '[.data[].type_de] | group_by(.) | map({v: .[0], n: length})'

# 2. voting 103928 — 52 members reported absent
curl -s "https://api.openparldata.ch/v1/votings/103928/votes?limit=500&lang_format=flat" \
  | jq '[.data[] | .vote + " / " + (.vote_display_de|tostring)] | group_by(.) | map({k: .[0], n: length})'

# 3. recapp has the type, but not the absent split
curl -s "https://zh.recapp.ch/viewer/api/shareparl/segments?agendaItemUid=c2c4b880-e83b-4ecc-aadb-5895d0f80f13&ios=false&language=de" \
  | jq -r '.[] | select(.title|startswith("Abstimmung"))
           | [.voting.voting_uid, .votingScheme, .voting.voting_type, .voting.protocol] | @tsv'
```

The official PDF for the 11:42 vote — named by recapp as `voting.protocol` — is
`VOTE-KRZH-20260615-114232-T007-00-0020554.pdf`, linked from the
[Geschäft page](https://www.kantonsrat.zh.ch/geschaefte/geschaeft/?id=89ddd67395d74b70bb1015edac49b7e2).

---

## Report A — as filed

> **Title:** Kanton Zürich (`body_key=ZH`): `votings.type_de` is null for every voting

First, thank you for OpenParlData — the harmonisation is the only reason covering a
second parliament is tractable for a small project at all.

I ran into this while building [zuerichratsinfo](https://github.com/SiiiTschiii/zuerichratsinfo),
a bot that publishes Zurich parliamentary vote results, which is live for Stadt Zürich
and preparing to cover the Kantonsrat. Measured against the live API on 2026-08-08.

### `votings.type_de` is null for every ZH voting

Sampling 200 recent votings per body:

| field | Kanton Zürich (`ZH`) | Stadt Zürich (`261`) |
| --- | --- | --- |
| `type_de` | null ×200 | `Normal` ×187, `Quorum` ×11, `Gleichgerichtete Anträge …` ×2 |

```bash
curl -s "https://api.openparldata.ch/v1/votings/?body_key=ZH&limit=200&lang_format=flat" \
  | jq '[.data[].type_de] | group_by(.) | map({v: .[0], n: length})'
```

The canton does record the distinction. recapp's segments payload — the same response
`stg_load_canton_insert_protocols_segements_ZH.hpl` already reads — carries it three
times over, keyed by `voting_uid`, which is the `external_id` the API already exposes:

```bash
curl -s "https://zh.recapp.ch/viewer/api/shareparl/segments?agendaItemUid=c2c4b880-e83b-4ecc-aadb-5895d0f80f13&ios=false&language=de" \
  | jq -r '.[] | select(.title|startswith("Abstimmung"))
           | [.voting.voting_uid, .votingScheme, .voting.voting_type] | @tsv'
```

```
C73B20F8-BE10-9CC8-70BA-B3510FCBA125	binary	0
1E159CE0-5FF2-6550-3DD8-6AC38D15BECC	quorum	4
D8C48612-302B-0CEB-C7C7-A8474CDD2C21	binary	0
A8D4D59E-0756-D4F6-AE63-D6CCAD573A79	quorum	4
5D0CFBDB-1691-F36C-1C75-48CE4002036C	quorum	4
```

The segment `title` says the same in plain German (`Abstimmung` vs
`Abstimmung Ausgabenbremse`), and the official PDF is headed `Geschäft: Quorum ermitteln`.

The pipeline already extracts `$.*.extVotingUid`, `$.*.title`, `$.*.type` and
`$.*.voting` from that response. It does not extract `$.*.votingScheme` — but
`voting_type` sits inside the `$.*.voting` blob that is staged, so this may be a matter
of propagating something already held rather than fetching anything new. Since `261`
already populates `type_de` (`stg_load_city_insert_votes_261_Zürich.hpl` has a
`voting_type_de` field and the ZH pipeline has no equivalent), this reads as an
oversight rather than a design decision.

**Why it matters:** an Ausgabenbremse (spending-brake) vote produces a lopsided tally —
129 Ja / 0 Nein / ~51 not voting. Republished without a type label it reads as
near-unanimous political agreement, when it is a procedural quorum vote most of the
opposition deliberately sits out. It cannot be derived from the tally either: voting
`104490` is 167:0 with 13 absent and is an ordinary approval.

### Reproduction

```bash
curl -s "https://api.openparldata.ch/v1/votings/?body_key=ZH&limit=200&lang_format=flat" \
  | jq '[.data[].type_de] | group_by(.) | map({v: .[0], n: length})'
# => [{"v":null,"n":200}]
```

Item page for one of the quorum votes: <https://openparldata.ch/item/votings/103928>

I have filed a second, separate issue about the same importer, covering `results_absent`
on ZH votings (#179) — unrelated cause, so it seemed better kept apart than bundled here.

---

## Report B — as filed

> **Title:** Kanton Zürich (`body_key=ZH`): `results_absent` conflates "nicht abgestimmt" with "nicht anwesend"

First, thank you for OpenParlData — the harmonisation is the only reason covering a
second parliament is tractable for a small project at all.

I ran into this while building [zuerichratsinfo](https://github.com/SiiiTschiii/zuerichratsinfo),
a bot that publishes Zurich parliamentary vote results, which is live for Stadt Zürich
and preparing to cover the Kantonsrat. Measured against the live API on 2026-08-08.

### `results_absent` conflates "did not vote" with "was not present"

Voting `103928` (`5D0CFBDB-1691-F36C-1C75-48CE4002036C`, 15.06.2026 11:42 local):

- **API:** `results_absent: 52`, and all 52 member records carry `vote: "absent"` /
  `vote_display_de: "Abwesend"` (128 `yes`, 180 records total).
- **Official record** (`VOTE-KRZH-20260615-114232-T007-00-0020554.pdf`, linked from the
  [Geschäft page](https://www.kantonsrat.zh.ch/geschaefte/geschaeft/?id=89ddd67395d74b70bb1015edac49b7e2)):
  header reads `Quorum 128`, **`Nicht anwesend 6`**, `Abgestimmt 128`. The individual
  table uses three values — `Quorum`, `Nicht abgestimmt`, `Nicht anwesend`. With a
  180-seat chamber that puts **46 members at `Nicht abgestimmt`** (180 − 6 − 128;
  derived, the PDF prints the split per name rather than as a subtotal).

So 46 members who were present and chose not to participate are published as absent.
For a spending-brake vote that choice is the political act, not an attendance record.

**One caveat on where the fix can come from:** recapp's segments payload does *not*
carry this distinction. Its per-member codes are `y` / `n` / `x` only, and for this
voting it reports `x ×52` — the same collapsed figure. The split exists only in the
PDF. `stg_load_canton_extract_pdf.hpl` and `stg_load_canton_votes_get_pdf_sub_ZH.hwf`
suggest the PDF path already exists in the pipeline, and recapp even names the file in
`voting.protocol`; but the per-member pattern in `stg_load_canton_insert_votes_ZH.hpl` —

```
^(.*?)\s(--|Ja|Nein|Enthaltung|Enthalten|enthalten|JA|NEIN|-|absent|ENTHALTEN)[^a-z].*
```

— has no branch for `Quorum`, `Nicht abgestimmt` or `Nicht anwesend`, which would
explain the collapse.

I appreciate that a new harmonised vote value is a schema change and not a small ask.
Even documenting that ZH `absent` means "did not cast a vote, for any reason" would let
downstream consumers stop asserting attendance they cannot support.

### Reproduction

```bash
curl -s "https://api.openparldata.ch/v1/votings/103928/votes?limit=500&lang_format=flat" \
  | jq '[.data[] | .vote + " / " + (.vote_display_de|tostring)] | group_by(.) | map({k: .[0], n: length})'
# => [{"k":"absent / Abwesend","n":52},{"k":"yes / Ja","n":128}]
```

Item page: <https://openparldata.ch/item/votings/103928>

I have filed a second, separate issue about the same importer, covering `votings.type_de`
being null for ZH (#178) — unrelated cause, so it seemed better kept apart than bundled here.

---

## Report C — as filed

> **Title:** Kanton Zürich (`body_key=ZH`): Cup-Abstimmung votings carry no aggregate results, and multi-option votes collapse under the harmonised `vote` value

First, thank you for OpenParlData — and for the quick turnaround on #178, which is
already live and working.

I ran into this while building [zuerichratsinfo](https://github.com/SiiiTschiii/zuerichratsinfo),
a bot that publishes Zurich parliamentary vote results, which is live for Stadt Zürich
and the Kantonsrat. Found while adding support for the `type_de` values #178 introduced.
Measured against the live API on 2026-08-09.

### 1. `Cup-Abstimmung` votings have null aggregate results

All four `Cup-Abstimmung` votings in the recent ZH window report
`results_yes`, `results_no` and `results_abstention` as **null**, with only
`results_absent` populated — even though the per-member records exist and are complete.

```bash
curl -s "https://api.openparldata.ch/v1/votings/?body_key=ZH&limit=200&lang_format=flat" \
  | jq -r '.data[] | select(.type_de=="Cup-Abstimmung")
           | [(.id|tostring), (.results_yes|tostring), (.results_no|tostring),
              (.results_abstention|tostring), (.results_absent|tostring)] | @tsv'
```

```
100011	null	null	null	8
98762	null	null	null	5
98752	null	null	null	5
98765	null	null	null	5
```

Meanwhile voting `98762`'s own `/votes` shows a complete result — 88 `Auswahl A`,
87 `Auswahl C`, 5 absent. So the data is there per member but never aggregated.

This looks like the same class of ZH-importer omission as #178, because **Stadt Zürich
populates aggregates for the equivalent vote type**:

```bash
curl -s "https://api.openparldata.ch/v1/votings/?body_key=261&limit=200&lang_format=flat" \
  | jq -r '.data[] | select(.type_de|startswith("Gleichgerichtete"))
           | [(.id|tostring), .type_de, (.results_yes|tostring)+"/"+(.results_no|tostring)+"/"+(.results_abstention|tostring)] | @tsv'
```

```
102150	Gleichgerichtete Anträge mit 3 Optionen	77/37/0
102149	Gleichgerichtete Anträge mit 4 Optionen	50/27/36
```

**Why it matters to us:** with every count null there is no result to render, so these
votings are rejected before publication and the Steuerfuss decision — three knockout
rounds on 15.12.2025 — goes unreported.

### 2. The harmonised `vote` value is lossy for multi-option votes

A secondary point, and possibly working as designed, but it is a trap for anyone
following the documented advice to read the harmonised `vote` rather than the display
string. For ZH multi-option votes the mapping is:

| `vote_display_de` | harmonised `vote` |
| --- | --- |
| Auswahl A | `yes` |
| Auswahl B | `no` |
| Auswahl C | `abstention` |
| Auswahl D | `further_option` |
| Auswahl E | `further_option` |

Two independent problems follow. `Auswahl D` and `Auswahl E` are **not distinguishable**
from the harmonised value — voting `98765` has A=88, C=59, D=23, E=5, which reads as
`further_option ×28`. And A/B/C map onto `yes`/`no`/`abstention`, so a consumer that
harmonises correctly renders a four-way choice as an ordinary Ja/Nein vote with
abstentions — silently, with no signal that the vote was multi-option.

```bash
curl -s "https://api.openparldata.ch/v1/votings/98765/votes?limit=500&lang_format=flat" \
  | jq -r '[.data[] | .vote + " / " + (.vote_display_de|tostring)] | group_by(.)
           | map(.[0] + " ×" + (length|tostring)) | join(", ")'
# => absent / Abwesend ×5, abstention / Auswahl C ×59, further_option / Auswahl D ×23,
#    further_option / Auswahl E ×5, yes / Auswahl A ×88
```

Either additional harmonised values, or a documented note that multi-option votes must
be read from `vote_display_de`, would be enough. `type_de` now makes such votes
identifiable up front, which helps a lot.

Filed separately from #178 and #179, which cover the same importer but unrelated causes.

---

## Report D — as filed

> **Title:** Kanton Zürich (`body_key=ZH`): `votings.decision` is null for every voting, though recapp publishes `votingResult`

Thank you again for the quick fix on #178 — `type_de` is live and working for us.

I ran into this while building [zuerichratsinfo](https://github.com/SiiiTschiii/zuerichratsinfo),
a bot that publishes Zurich parliamentary vote results, live for Stadt Zürich and the
Kantonsrat. Measured against the live API on 2026-08-09.

### `decision` is null for every ZH voting

Sampling 200 recent votings per body:

```bash
curl -s "https://api.openparldata.ch/v1/votings/?body_key=ZH&limit=200&lang_format=flat" \
  | jq '[.data[].decision] | group_by(.) | map({v: .[0], n: length})'
# => [{"v":null,"n":200}]

curl -s "https://api.openparldata.ch/v1/votings/?body_key=261&limit=200&lang_format=flat" \
  | jq '[.data[].decision] | group_by(.) | map({v: .[0], n: length})'
# => Ja ×169, Nein ×29, "Auswahl A" ×1, null ×1
```

### recapp publishes the outcome

The same segments response `stg_load_canton_insert_protocols_segements_ZH.hpl` already
reads carries `votingResult` alongside the `votingScheme` that #178 adopted, keyed by
the same `voting_uid`:

```bash
curl -s "https://zh.recapp.ch/viewer/api/shareparl/segments?agendaItemUid=c2c4b880-e83b-4ecc-aadb-5895d0f80f13&ios=false&language=de" \
  | jq -r '.[] | select(.title|startswith("Abstimmung"))
           | [.voting.voting_uid, .votingScheme, .votingResult] | @tsv'
```

```
C73B20F8-BE10-9CC8-70BA-B3510FCBA125	binary	yes
1E159CE0-5FF2-6550-3DD8-6AC38D15BECC	quorum	yes
D8C48612-302B-0CEB-C7C7-A8474CDD2C21	binary	yes
A8D4D59E-0756-D4F6-AE63-D6CCAD573A79	quorum	yes
5D0CFBDB-1691-F36C-1C75-48CE4002036C	quorum	yes
```

The pipeline extracts no `votingResult` (`grep -c votingResult` over that `.hpl`
returns 0) — the same shape of omission as `votingScheme` before #178.

### Why this is not simply derivable downstream

We had been deriving the outcome from `results_yes > results_no`, and want to flag that
this is unsafe, in case other consumers do the same. Comparing Ja against Nein only
decides an outcome when they are the two sides of one question. In a **quorum** vote
they are not: there is no Nein to cast, so `results_no` is structurally 0 and every such
vote derives as accepted.

Concretely, ZH voting `100969` — a quorum vote with 41 supporters and 139 not supporting — derived
as "angenommen" and would have been published that way. Whether it actually passed
depends on a threshold that is not in the API, not in recapp, and not in the official
PDF, and which differs by the procedure the vote serves — `type_de` says `Quorum` for
Ausgabenbremse, initiative support, and Wahlen alike, at visibly different levels:

| shape | supporters | example |
| --- | --- | --- |
| Ausgabenbremse | 128–129 | Staatsbeitrag Glattalbahn, Mindereinnahmenbremse |
| Initiative support | 0–116 | Verbot biometrischer Gesichtserkennung (54), Energiegesetz 2050 (95) |
| Wahlen / Mitteilungen | 153–174 | Wahl Mitglied Obergericht |

We have stopped deriving it and now publish cantonal votes with no outcome line at all,
which is a visible regression for readers but the only honest option. `votingResult`
would restore it.

Filed separately from #178, #179 and the Cup-Abstimmung issue, which cover the same
importer but unrelated causes.
