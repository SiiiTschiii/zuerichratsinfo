# Zurich Ratsinfo

<p align="center">
  <img src="assets/logo.svg" alt="Zurich Ratsinfo Logo" width="200"/>
</p>

A civic tech bot that shares updates from the Zurich City Council (Gemeinderat Zürich) on social media platforms and tags relevant politicians based on a curated list of their social media accounts.

## Supported Platforms

| Platform    | Status     | Account                                           |
| ----------- | ---------- | ------------------------------------------------- |
| X (Twitter) | ✅ Active  | [@zuerichratsinfo](https://x.com/zuerichratsinfo) |
| Facebook    | ❌ Planned | -                                                 |
| Instagram   | ❌ Planned | -                                                 |
| LinkedIn    | ❌ Planned | -                                                 |
| Bluesky     | ❌ Planned | -                                                 |
| TikTok      | ❌ Planned | -                                                 |

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

## License

MIT
