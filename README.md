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

A civic tech bot that shares updates from the Zurich City Council (Gemeinderat Zürich) on social media platforms and tags relevant politicians based on a curated list of their social media accounts.

## Supported Platforms

| Platform    | Status     | Gemeinderäte | Account                                           |
| ----------- | ---------- | ------------ | ------------------------------------------------- |
| LinkedIn    | ❌ Planned | 86 | -                                                 |
| Facebook    | ❌ Planned | 77 | -                                                 |
| Instagram   | ❌ Planned | 71 | -                                                 |
| X (Twitter) | ✅ Active  | 63 | [@zuerichratsinfo](https://x.com/zuerichratsinfo) |
| Bluesky     | ❌ Planned | 25 | -                                                 |
| TikTok      | ❌ Planned | 13 | -                                                 |

_Platforms are sorted by coverage. Out of 126 total contacts in [data/contacts.yaml](data/contacts.yaml)._

## What It Does

- **Automated Vote Posts**: Shares council vote results (Abstimmungen) from the [Gemeinderat Zürich](https://www.gemeinderat-zuerich.ch/) on social media platforms
- **Politician Tagging**: Automatically tags mentioned politicians using their social media accounts when available in our mapping
- **Social Media Mapping**: Curates an extensive mapping of Zurich politicians to their social media accounts (X, Facebook, Instagram, LinkedIn, Bluesky, TikTok) - see [data/contacts.yaml](data/contacts.yaml)

### Contributing to the Social Media Mapping

Found an error or want to add a politician's social media account? Please [open an issue](https://github.com/SiiiTschiii/zuerichratsinfo/issues/new) or submit a pull request!

## Tech Stack

- Go
- Zurich Council PARIS API, see [pkg/zurichapi/README.md](pkg/zurichapi/README.md)
- X API v2 with OAuth 1.0a, see [pkg/xapi/README.md](pkg/xapi/README.md)
- is.gd API for URL shortening

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

## License

MIT
