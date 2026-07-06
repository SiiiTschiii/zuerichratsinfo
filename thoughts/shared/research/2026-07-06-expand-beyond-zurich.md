---
date: 2026-07-06T14:00:00+02:00
researcher: claude
topic: "Expand beyond Zurich city politics — data source feasibility"
tags: [research, expansion, kantonsrat, bund, openparldata, apis]
status: complete
last_updated: 2026-07-06
---

# Research: Expand Beyond Zurich City Politics

**Date**: 2026-07-06
**TODO item**: "Expand beyond Zurich city politics — other Swiss cities, cantonal parliaments/governments, federal level, potentially other countries with suitable public data APIs"

## Research Question

Which jurisdictions beyond the City of Zurich have public data sources good enough to power the same bot concept (automated posts of council vote results with per-member/per-Fraktion breakdown and politician tagging)? Feasibility only — no implementation.

## What a New Jurisdiction Must Provide

Derived from what the bot actually consumes today (`pkg/zurichapi`, `pkg/voteposting`):

1. **Vote results via API** — machine-readable (not PDF), ideally polled hourly like PARIS
2. **Per-member votes (roll call)** — needed for the per-Fraktion breakdown replies and politician tagging; aggregate Ja/Nein alone supports only a reduced format
3. **Stable vote IDs** — for the `data/posted_votes_*.json` dedup log
4. **Business (Geschäft) metadata** — title, type, submitters, date, for grouping votes into posts
5. **Member data** — names/party/Fraktion, to seed a per-jurisdiction `contacts.yaml`
6. **Timeliness** — PARIS publishes ~5–7 days after the vote; anything in that range or better is fine
7. **Open license / bot-friendly terms** — attribution-style licenses (CC BY, OGD terms)

Non-data prerequisites that apply to *every* expansion (often the bigger cost):

- New social media accounts per jurisdiction (X Premium cost per account, Meta app/page setup, Bluesky handle)
- A curated contacts mapping per jurisdiction — this is the project's differentiator and is manual work (Zurich: 132 contacts, curated over months)
- Language (French/Italian for Romandie/Ticino; local language abroad)
- GitHub Actions scheduling and secrets multiply per jurisdiction

## Architecture Note

The pipeline is currently hard-coupled to the Zurich source: `zurichapi.Abstimmung`/`Stimmabgabe` types are referenced in **15 files / ~125 places** across `pkg/voteposting`, all three platform formatters, and `pkg/imagegen`. Any expansion needs a source-neutral domain model (Vote, MemberVote, Business, Member) with per-source adapters first. That refactor is jurisdiction-independent and is the real first implementation step whenever expansion starts.

---

## Tier 1a: Federal Level (Bund) — HIGH feasibility, best data in Switzerland

**Source**: Official Swiss Parliament web services, `https://ws.parlament.ch/odata.svc/` (OData, JSON/XML, no auth). Legacy REST at `ws-old.parlament.ch` (has a dedicated `votes/` resource). Documented at [parlament.ch Open Data / Web Services](https://www.parlament.ch/de/%C3%BCber-das-parlament/fakten-und-zahlen/open-data-web-services).

- **Roll-call votes**: `Voting` table has the individual vote of every Nationalrat member for every electronic vote since the 48th legislature (winter session 2007) — same shape as PARIS `Stimmabgaben` (person, party, Fraktion, Ja/Nein/Enthaltung/Abwesend). The [Abstimmungs-Datenbank](https://www.parlament.ch/de/ratsbetrieb/abstimmungen/abstimmungs-datenbank-nr) exposes the same data with exports.
- **Ständerat caveat**: electronic voting since spring 2014, but name lists are only published for final votes, constitutional/urgency votes, or when 10+ members request it. Coverage is partial by design. (Verify current rules — there have been ongoing transparency pushes.)
- **Everything else**: `Business`, `Session`, `MemberCouncil`, `Person`, committees, transcripts — covers Geschäft metadata and member seeding completely.
- **Ecosystem**: mature — R package [`swissparl`](https://github.com/zumbov2/swissparl), Python [`swissparlpy`](https://github.com/metaodi/swissparlpy) (by the same Open Data Zurich person who supported the PARIS integration — existing contact!).
- **Timeliness**: votes appear during/shortly after sessions — *faster* than PARIS.
- **Risks**: sessions are 4×3 weeks/year → very bursty posting volume (dozens of votes/day in session, then silence); would need digest/threshold logic (e.g. only Schlussabstimmungen and Gesamtabstimmungen). Terms of use require source attribution; fine for this project.
- **Audience fit**: largest possible Swiss audience; crowded space though (SRF, smartvote/smartmonitor, Nau, etc. already report NR votes — the per-Fraktion + tagging angle is still differentiated).

**Verdict**: technically the easiest and highest-quality expansion. Main costs are editorial (vote selection policy) and contacts curation (246 members, though federal politicians' social handles are comparatively easy to find and partially exist in public datasets).

## Tier 1b: Canton Zürich (Kantonsrat) — MEDIUM feasibility, data gap on roll calls

**Source**: official web service of the Kantonsrat Geschäftsverwaltungssystem (CMI Axioma), endpoints under `https://parlzhcdws.cmicloud.ch/...` (XML), documented via [opendata.swiss dataset](https://opendata.swiss/de/dataset/web-service-des-geschaftsverwaltungssystems-des-kantonsrates-des-kantons-zurich) ("Organisation und Geschäfte des Zürcher Kantonsrats").

- **Available via XML web service**: Geschäfte, session agendas (Traktanden), current members by Partei/Fraktion/Kommission, Kantonsrat dispatch documents, protocols since May 1995, Gremien.
- **The gap**: per-member vote results. The Kantonsrat votes electronically and voting behavior is public, but the published *Abstimmungsprotokolle* are downloadable per session from kantonsrat.zh.ch **as PDFs** — the opendata.swiss dataset description does not list an Abstimmungen endpoint. Options if confirmed:
  - PDF parsing of Abstimmungsprotokolle (fragile, but format is uniform per legislature)
  - **OpenParlData** (below) may already have harmonized KR-ZH votes — this is the single most important thing to verify
  - Ask the Parlamentsdienste / OGD Fachstelle Kanton Zürich (the project has good standing with the city's OGD team; the canton has an active OGD office and `statistikZH` GitHub org) whether a vote endpoint exists or can be added
- **Audience fit**: best strategic fit — same media market, same parties, overlapping politicians (many Kantonsräte are ex-Gemeinderäte), German only, and an obvious brand extension (@zuerichratsinfo → canton).

**Verdict**: strongest audience synergy; blocked only on machine-readable roll-call data. Verification of the cdws endpoints and OpenParlData coverage is the concrete next step.

## Strategic Option: OpenParlData.ch — potentially changes the whole roadmap

[OpenParlData](https://opendata.ch/projects/openparldata/) (live since **2026-02-09**; Glue + Opendata.ch + Stiftung Mercator Schweiz) harmonizes parliamentary data from the **Bund, all 26 cantons, and 50+ cities** (78 systems) into one REST API ([Swagger docs](https://api.openparldata.ch/documentation)), updated daily, **CC BY 4.0**, with bulk exports. Entities include persons, affairs, memberships, sessions, **votings/votes**, interests, speeches.

- Endorsed by ecosystem moves already: `swissparl` (R) and [`swissparlpy`](https://metaodi.ch/swissparlpy/) added OpenParlData support; Liip [migrated their Swiss Parliament Bot to it](https://www.liip.ch/en/blog/new-api-new-scope-new-mcp-server-upgrading-the-swiss-parliament-bot) and now covers "any parliament in the system".
- **If** its votings data includes per-member ballots for Kantonsrat ZH and other cantons/cities, the bot could build **one** OpenParlData adapter and get most Swiss jurisdictions nearly for free — instead of one adapter per CMI/Axioma instance.
- **Risks**: beta ("Ecken und Kanten" per launch coverage), young project, unknown depth/latency per parliament, sustainability depends on foundation funding. Underlying reality doesn't change: parliaments that only publish vote PDFs won't magically have per-member data here either. Keep direct-source adapters as fallback for flagship jurisdictions.

**Verdict**: evaluate before building any per-canton adapter. Concretely: query `/votings` for Kanton ZH and City of Zurich, compare with PARIS data for completeness and lag.

## Tier 2: Other Swiss Cities and Cantons

| Jurisdiction | Source | Per-member votes? | Feasibility |
|---|---|---|---|
| **Basel-Stadt (Grosser Rat)** — canton, acts as city too | [data.bs.ch dataset 100186](https://data.bs.ch/explore/dataset/100186/) "Live-Abstimmungsergebnisse" + Geschäfte/Vorstösse datasets (Opendatasoft API, JSON/CSV) | **Yes — per member, updated in real time on session days** | **HIGH — best sub-federal data in CH** |
| **Kanton Bern (Grosser Rat)** | ["Votes du Grand Conseil / Abstimmungsresultate GR"](https://www.i14y.admin.ch/fr/catalog/datasets/KTBE-APM0002162-AbstimmungsresultateGR-V1) on i14y; machine-readable session protocols since winter 2022 | Likely yes (dataset exists; structure to verify) | MEDIUM-HIGH |
| **Stadt Bern (Stadtrat)** | [Bern OGD](https://www.bern.ch/open-government-data-ogd/ogd-nach-themen/stadtrat-ogd): machine-readable agendas + decisions; RIS website | Electronic voting exists, but vote protocols published as **PDF annex** to session protocol | MEDIUM-LOW |
| **Stadt St. Gallen (Stadtparlament)** | [RIS dataset on daten.stadt.sg.ch](https://daten.stadt.sg.ch/explore/dataset/traktandierte-geschaefte-sitzungen-stadtparlament-stgallen/api/) (Opendatasoft) | Agenda items only; no vote dataset found | LOW-MEDIUM |
| **Winterthur (Stadtparlament)** | parlament.winterthur.ch; city OGD portal still in build-out | Nothing found | LOW (watch OpenParlData) |
| **Luzern, Lausanne, Genève (city)** | No vote APIs surfaced in search | Unknown | Unknown — check via OpenParlData; FR language cost for Romandie |
| **Kanton Aargau (Grosser Rat)** | Beschlussprotokoll published right after each session (documents) | Not as open data | LOW |

Pattern: most Swiss councils run CMI/Axioma-family systems that expose Geschäfte/agenda/members in XML but keep vote results in PDF protocols. Basel-Stadt is the standout exception (and covers canton + city in one). This is exactly the fragmentation OpenParlData targets.

## Tier 3: Other Countries

Ranked by data quality for a vote bot; all carry the full non-data cost (new accounts, new contacts curation, language, no brand recognition):

- **European Parliament — HIGH (data), natural next audience for a multi-lingual project.** [HowTheyVote.eu](https://howtheyvote.eu/) provides cleaned roll-call votes + MEP data, [weekly-updated open data on GitHub](https://github.com/HowTheyVote/data) (CSV, open license); primary source is the EP open data portal. Roll-call votes are per-MEP. Caveat: many EP votes are not roll-call; a Swiss-focused project has no obvious audience angle here.
- **Germany — HIGH.** Bundestag publishes namentliche Abstimmungen as machine-readable lists ([bundestag.de Open Data](https://www.bundestag.de/services/opendata), XLSX/XML per vote) plus DIP API for Drucksachen; [abgeordnetenwatch.de API](https://www.abgeordnetenwatch.de/api) is the better integration point: CC0, JSON, `vote` entity with per-MP behavior, covers Bundestag **and Landtage + EU**, 30 req/min. Caveat: only ~50–100 namentliche Abstimmungen/year (most Bundestag votes are by show of hands); abgeordnetenwatch itself already occupies much of this civic-tech niche. Municipal level (OParl standard, many Ratsinformationssysteme) has business/agenda data but essentially never per-member votes.
- **UK — HIGH.** Official [Commons Votes / Lords Votes APIs](https://developer.parliament.uk/) (divisions as JSON, every division is per-member by definition) + Members API; extremely bot-friendly. Note current API URL migration (old commonsvotes-api URLs deprecated March 2026). TheyWorkForYou already serves this niche well.
- **Netherlands — MEDIUM.** [Tweede Kamer OData API](https://opendata.tweedekamer.nl/) with `Stemming` entity — but Dutch votes are recorded **per party** (fractie), not per member, except rare hoofdelijke stemmingen. Party-level matches the bot's Fraktion-breakdown format, not the tagging format.
- **US — MEDIUM-HIGH.** [Congress.gov API](https://github.com/LibraryOfCongress/api.congress.gov) added House roll-call endpoints (beta, May 2025) incl. member votes; Senate roll calls as XML. API key required. Very crowded space.
- **Austria — LOW.** [Parliament open data API](https://www.parlament.gv.at/recherchieren/open-data) (JSON) is good for documents/protocols, but Nationalrat votes are mostly by show of hands; namentliche Abstimmungen are rare and live in stenographic protocols, not structured vote data. The core product (roll-call posts) doesn't work here.

## Recommendation (order of expansion)

1. **Refactor first** (whenever expansion starts): extract a source-neutral vote/member domain model out of `zurichapi` types; make `contacts.yaml`, posted-votes logs, and account credentials per-jurisdiction.
2. **Evaluate OpenParlData** (days, no commitment): check `/votings` coverage/lag for Kanton ZH, City of Zurich (cross-check against PARIS), Bund. Outcome decides between "one harmonized adapter" and "per-source adapters".
3. **Federal Nationalrat** as the first new jurisdiction: best official data (OData or OpenParlData), biggest audience, German-first works, existing ecosystem contacts. Needs an editorial policy for session bursts (e.g. Schlussabstimmungen only).
4. **Kantonsrat Zürich** next for brand synergy — via OpenParlData if its vote data checks out, else contact Parlamentsdienste ZH / OGD Fachstelle about a machine-readable Abstimmungen endpoint (PDF parsing only as last resort).
5. **Basel-Stadt** as the best-data opportunistic add (real-time per-member API, canton+city in one) — if there's appetite for a non-Zurich brand (@baselratsinfo).
6. **International** (EU Parliament or Germany first) only after the multi-jurisdiction machinery is proven in Switzerland; each country is effectively a new product (accounts, contacts, language, competition from established players like abgeordnetenwatch/TheyWorkForYou).

## Open Questions / To Verify (blocked in this session)

This session's sandbox egress policy blocked direct API calls (only web search was available), so the following need hands-on verification from a normal network:

- [ ] OpenParlData: does `/votings` contain per-member ballots for Kantonsrat ZH / Stadt Zürich / Bund? Freshness vs PARIS? Beta stability, rate limits?
- [ ] Kantonsrat ZH cdws web service: enumerate actual endpoints (`parlzhcdws.cmicloud.ch/.../cdws/...`) — is there any Abstimmungen resource beyond the opendata.swiss description?
- [ ] ws.parlament.ch OData: confirm `Voting` publication lag during sessions; current Ständerat name-list rules
- [ ] Basel-Stadt dataset 100186: field structure, historical depth, Opendatasoft rate limits
- [ ] Kanton Bern i14y dataset KTBE-APM0002162: per-member or aggregate?
- [ ] parlament.ch terms of use details (attribution wording for social posts)

## Sources

- [opendata.swiss — Web Service Geschäftsverwaltungssystem Kantonsrat ZH](https://opendata.swiss/de/dataset/web-service-des-geschaftsverwaltungssystems-des-kantonsrates-des-kantons-zurich)
- [Kantonsrat Zürich](https://www.kantonsrat.zh.ch/) · [Geschäfte](https://www.kantonsrat.zh.ch/geschaefte/)
- [parlament.ch — Open Data / Web Services](https://www.parlament.ch/de/%C3%BCber-das-parlament/fakten-und-zahlen/open-data-web-services) · [Abstimmungs-Datenbank NR](https://www.parlament.ch/de/ratsbetrieb/abstimmungen/abstimmungs-datenbank-nr) · [ws-old.parlament.ch](https://ws-old.parlament.ch/)
- [swissparl (R)](https://github.com/zumbov2/swissparl) · [swissparlpy (Python)](https://github.com/metaodi/swissparlpy)
- [OpenParlData — Projektseite](https://opendata.ch/projects/openparldata/) · [API-Doku (Swagger)](https://api.openparldata.ch/documentation) · [Glue launch post](https://www.glue.ch/en/2026/02/09/openparldata-goes-live/) · [IT-Markt Bericht](https://www.it-markt.ch/news/2026-02-17/neue-plattform-buendelt-politische-daten-der-schweiz) · [Liip: Swiss Parliament Bot on OpenParlData](https://www.liip.ch/en/blog/new-api-new-scope-new-mcp-server-upgrading-the-swiss-parliament-bot)
- [data.bs.ch — Grosser Rat Live-Abstimmungsergebnisse (100186)](https://data.bs.ch/explore/dataset/100186/) · [grosserrat.bs.ch Abstimmungsergebnisse](https://grosserrat.bs.ch/ratsbetrieb/tagesordnung/abstimmungsergebnisse)
- [i14y — Votes du Grand Conseil (BE)](https://www.i14y.admin.ch/fr/catalog/datasets/KTBE-APM0002162-AbstimmungsresultateGR-V1)
- [Stadt Bern — Stadtrat OGD](https://www.bern.ch/open-government-data-ogd/ogd-nach-themen/stadtrat-ogd)
- [daten.stadt.sg.ch — RIS Stadtparlament St.Gallen](https://daten.stadt.sg.ch/explore/dataset/traktandierte-geschaefte-sitzungen-stadtparlament-stgallen/api/)
- [Stadtparlament Winterthur](https://parlament.winterthur.ch/) · [Stadt Winterthur Open Data](https://stadt.winterthur.ch/themen/die-stadt/winterthur/statistik/open-data)
- [Kanton Aargau — Traktandenliste & Protokolle](https://www.ag.ch/de/ueber-uns/grosser-rat/traktandenliste-protokolle)
- [Bundestag Open Data](https://www.bundestag.de/services/opendata) · [Namentliche Abstimmungen](https://www.bundestag.de/parlament/plenum/abstimmung/abstimmungen) · [abgeordnetenwatch API](https://www.abgeordnetenwatch.de/api) · [Vote-Entität](https://www.abgeordnetenwatch.de/api/entitaeten/vote)
- [UK Parliament Developer Hub](https://developer.parliament.uk/) · [Commons Votes API](https://commonsvotes-api.parliament.uk/swagger/ui/index) · [Members API](https://members-api.parliament.uk/index.html)
- [HowTheyVote.eu](https://howtheyvote.eu/) · [HowTheyVote/data](https://github.com/HowTheyVote/data)
- [Tweede Kamer Open Data Portaal](https://opendata.tweedekamer.nl/) · [OData API](https://opendata.tweedekamer.nl/documentatie/odata-api)
- [Congress.gov API (GitHub)](https://github.com/LibraryOfCongress/api.congress.gov) · [House roll-call announcement](https://blogs.loc.gov/law/2025/05/introducing-house-roll-call-votes-in-the-congress-gov-api/)
- [Parlament Österreich Open Data](https://www.parlament.gv.at/recherchieren/open-data)
