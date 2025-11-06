# Zurich Ratsinfo

A civic tech bot that shares updates from the Zurich City Council (Gemeinderat Zürich) on X.

Follow [@zuerichratsinfo](https://x.com/zuerichratsinfo) for the latest council votes.

## What It Does

Automatically posts council vote results (Abstimmungen) from the [Gemeinderat Zürich](https://www.gemeinderat-zuerich.ch/) to X.

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
