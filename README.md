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
  <a href="https://goreportcard.com/report/github.com/siiitschiii/zuerichratsinfo">
    <img src="https://goreportcard.com/badge/github.com/siiitschiii/zuerichratsinfo" alt="Go Report Card">
  </a>
</p>

A civic tech bot that shares parliamentary vote results from Zurich on social media and tags the politicians involved, using a curated mapping of their accounts.

## Covered Bodies

| Body                                                          | Data source                                              | Status                        | Tagging               |
| ------------------------------------------------------------- | -------------------------------------------------------- | ----------------------------- | --------------------- |
| **Gemeinderat Stadt Zürich** (125 seats)                       | [PARIS API](https://opendatazurich.github.io/paris-api/), City of Zurich | ✅ Live | ✅ 132 curated contacts |
| **Kantonsrat Zürich** (180 seats)                              | [OpenParlData](https://api.openparldata.ch/documentation) | 🟡 Implemented, not yet enabled | ❌ Not yet curated |

Both bodies post to the same accounts. Every post names which chamber voted, in
the text and on the image, so the two are never confused.

## Supported Platforms

| Platform    | Status     | Gemeinderäte & Stadträte | Account                                                                              |
| ----------- | ---------- | ------------------------ | ------------------------------------------------------------------------------------ |
| LinkedIn    | ❌ Planned | 108 | -                                                                                    |
| Facebook    | ❌ Planned | 83 | -                                                                                    |
| Instagram   | ✅ Active  | 92 | [@zueriratsinfo](https://www.instagram.com/zueriratsinfo)                            |
| X (Twitter) | ✅ Active  | 59 | [@zuerichratsinfo](https://x.com/zuerichratsinfo)                                    |
| Bluesky     | ✅ Active  | 28 | [@zuerichratsinfo.bsky.social](https://bsky.app/profile/zuerichratsinfo.bsky.social) |
| TikTok      | ❌ Planned | 18 | -                                                                                    |

_Platforms are sorted by coverage. Counts include both Gemeinderäte and Stadträte (9 Stadtrat members). Out of 132 total contacts in [data/zurich-city/contacts.yaml](data/zurich-city/contacts.yaml). Kantonsrat members are not yet curated and so are not counted here._

## What It Does

- **Automated Vote Posts**: Shares vote results (Abstimmungen) with the full per-faction breakdown. Timing depends on when each source publishes: the [PARIS API](https://opendatazurich.github.io/paris-api/) typically 5–7 days after a city vote — the same data behind [gemeinderat-zuerich.ch](https://www.gemeinderat-zuerich.ch/sitzungen/termine/?navid=968842968842) — and OpenParlData within a few days of a cantonal one.
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

## Support This Project

Help keep @zuerichratsinfo running! Your support covers the costs for X Premium account and API access.

<a href="https://buymeacoffee.com/zuerichratsinfo" target="_blank"><img src="https://cdn.buymeacoffee.com/buttons/v2/default-yellow.png" alt="Buy Me A Coffee" style="height: 60px !important;width: 217px !important;" ></a>

This is a non-profit civic tech project. Every contribution helps make local politics more accessible! 🙏

## Acknowledgments

Special thanks to:

- **[Alexander Guentert](https://github.com/alexanderguentert)** from [Open Data Zurich](https://opendatazurich.github.io) for support in integrating the Paris-API, Gemeinderat Stadt Zürich
- **[OpenParlData](https://api.openparldata.ch)** for harmonised Swiss parliamentary data (CC BY 4.0), which is what makes the Kantonsrat — and any further canton — possible

## License

MIT
