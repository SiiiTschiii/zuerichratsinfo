# Zurich Ratsinfo

<p align="center">
  <img src="assets/logo.svg" alt="Zurich Ratsinfo Logo" width="200"/>
</p>

<p align="center">
  <a href="https://github.com/SiiiTschiii/zuerichratsinfo/actions/workflows/go-ci.yml">
    <img src="https://github.com/SiiiTschiii/zuerichratsinfo/actions/workflows/go-ci.yml/badge.svg" alt="Go CI">
  </a>
  <a href="https://github.com/SiiiTschiii/zuerichratsinfo/actions/workflows/validate-contacts.yml">
    <img src="https://github.com/SiiiTschiii/zuerichratsinfo/actions/workflows/validate-contacts.yml/badge.svg" alt="Validate Contacts">
  </a>
  <a href="https://github.com/SiiiTschiii/zuerichratsinfo/actions/workflows/bot.yml">
    <img src="https://img.shields.io/github/last-commit/SiiiTschiii/zuerichratsinfo/main?label=last%20bot%20run&logo=github" alt="Last Bot Run">
  </a>
</p>

A civic tech bot that shares parliamentary vote results from Zurich on social media and tags the politicians involved, using a curated mapping of their accounts.

We built it because a healthy democracy rests on two things that are easy to lose sight of:

- **Participation**: citizens who can actually follow and take part in the decisions that affect them.
- **Accountability**: what politicians actually do, from how they vote to which motions, postulates and petitions they put their name on, visible right where people already are.

**🇨🇭 Auf Deutsch**

Ein Civic-Tech-Bot, der Abstimmungsergebnisse aus den Zürcher Parlamenten auf Social Media teilt und die beteiligten Politikerinnen und Politiker markiert, anhand einer manuell gepflegten Zuordnung ihrer Accounts.

Wir haben das Projekt ins Leben gerufen, weil eine gesunde Demokratie auf zwei Dingen beruht, die leicht aus dem Blick geraten:

- **Partizipation**: Bürgerinnen und Bürger, die die Entscheide, die sie betreffen, tatsächlich mitverfolgen und mitgestalten können.
- **Rechenschaft**: was Politikerinnen und Politiker wirklich tun, vom Abstimmungsverhalten bis zu den Motionen, Postulaten und Petitionen, die sie mittragen, sichtbar dort, wo die Menschen ohnehin schon sind.

## Covered Bodies

| Body                         | Data source                                                              | Status  | Tagging                 |
| ---------------------------- | ------------------------------------------------------------------------ | ------- | ----------------------- |
| **Gemeinderat Stadt Zürich** | [PARIS API](https://opendatazurich.github.io/paris-api/), City of Zurich | ✅ Live | ✅ 132 curated contacts |
| **Kantonsrat Zürich**        | [OpenParlData](https://api.openparldata.ch/documentation)                | ✅ Live | ❌ Not yet curated      |

Both bodies post to the same accounts. Every post names which chamber voted, in
the text and on the image, so the two are never confused.

## Supported Platforms

| Platform    | Status     | Politicians with a verified account | Account                                                                              |
| ----------- | ---------- | ----------------------------------- | ------------------------------------------------------------------------------------ |
| LinkedIn    | ❌ Planned | 108                                 | -                                                                                    |
| Facebook    | ❌ Planned | 83                                  | -                                                                                    |
| Instagram   | ✅ Active  | 91                                  | [@zueriratsinfo](https://www.instagram.com/zueriratsinfo)                            |
| X (Twitter) | ✅ Active  | 59                                  | [@zuerichratsinfo](https://x.com/zuerichratsinfo)                                    |
| Bluesky     | ✅ Active  | 28                                  | [@zuerichratsinfo.bsky.social](https://bsky.app/profile/zuerichratsinfo.bsky.social) |
| TikTok      | ❌ Planned | 18                                  | -                                                                                    |

_"Politicians with a verified account" is how many Gemeinderat and Stadtrat members
(9 of them Stadträte) we have manually identified and verified as being active on
that platform, so that they can be tagged in a post, out of the 132 politicians
curated in [data/zurich-city/contacts.yaml](data/zurich-city/contacts.yaml).
Platforms are sorted by coverage. Kantonsrat members are not yet curated and so are
not counted here._

## What It Does

- **Automated Vote Posts**: Shares vote results (Abstimmungen) with the full per-faction breakdown. Timing depends on when each source publishes: the [PARIS API](https://opendatazurich.github.io/paris-api/) typically 5–7 days after a city vote (the same data behind [gemeinderat-zuerich.ch](https://www.gemeinderat-zuerich.ch/sitzungen/termine/?navid=968842968842)), and OpenParlData within a few days of a cantonal one.
- **Politician Tagging**: Automatically tags mentioned politicians using their social media accounts when available in our mapping
  - _Example: "Postulat von Ivo Bieri @ivo_bieri (SP) und Liv Mahrer @LivMahrer (SP)..."_
- **Social Media Mapping**: Curates an extensive mapping of Zurich politicians to their social media accounts (X, Facebook, Instagram, LinkedIn, Bluesky, TikTok) - see [data/zurich-city/contacts.yaml](data/zurich-city/contacts.yaml)

## Tech Stack

- Go
- Zurich Council PARIS API, see [pkg/zurichapi/README.md](pkg/zurichapi/README.md)
- OpenParlData API for the Kantonsrat, see [pkg/openparldata](pkg/openparldata)
- X API v2 with OAuth 1.0a, see [pkg/xapi/README.md](pkg/xapi/README.md)
- Bluesky AT Protocol (app.bsky), see [pkg/voteposting/platforms/bluesky](pkg/voteposting/platforms/bluesky)
- Instagram Graph API with image carousel publishing, see [pkg/igapi/README.md](pkg/igapi/README.md)
- Vote image generation (1080×1350 JPEG carousels), see [pkg/imagegen](pkg/imagegen)

## Setup

See [SETUP.md](SETUP.md) for installation and configuration instructions.

## Project Progress

See [TODO.md](TODO.md) for current tasks and roadmap.

## Recognition

zuerichratsinfo is featured on the [City of Zurich's Open Government Data portal](https://www.stadt-zuerich.ch/de/politik-und-verwaltung/statistik-und-daten/open-government-data/anwendungen/anwendungen-2026/zuerichratsinfo.html) as an exemplary application using public government data to improve democratic transparency.

## Support This Project

Help keep @zuerichratsinfo running! Your support covers the costs for X Premium account and API access.

<a href="https://buymeacoffee.com/zuerichratsinfo" target="_blank"><img src="https://cdn.buymeacoffee.com/buttons/v2/default-yellow.png" alt="Buy Me A Coffee" style="height: 60px !important;width: 217px !important;" ></a>

This is a non-profit civic tech project. Every contribution helps make local politics more accessible! 🙏

## Acknowledgments

Special thanks to:

- **[Alexander Guentert](https://github.com/alexanderguentert)** from [Open Data Zurich](https://opendatazurich.github.io) for support in integrating the Paris-API, Gemeinderat Stadt Zürich
- **[OpenParlData](https://opendata.ch/projects/openparldata/)** for harmonised Swiss parliamentary data (CC BY 4.0), which is what makes the Kantonsrat, and any further canton, possible ([API docs](https://api.openparldata.ch))

## License

MIT
