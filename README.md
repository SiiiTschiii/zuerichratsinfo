# Zurich Ratsinfo

A civic tech bot that shares updates from the Zurich City Council (Gemeinderat Zürich) on X.

Follow [@zuerichratsinfo](https://x.com/zuerichratsinfo) for the latest council submissions.

## What It Does

Automatically posts new council business (motions, written questions, proposals) from the [Gemeinderat Zürich](https://www.gemeinderat-zuerich.ch/) to X.

Example tweet:
```
🏛️ Neues Geschäft im Gemeinderat Zürich

📋 2025/459: Motion
📅 01.10.2025 von Anjushka Früh (SP)

Strategie zur Einforderung eines angemessenen Anteils...
```

## Tech Stack

- Go
- [Zurich Council PARIS API](https://opendatazurich.github.io/paris-api/)
- X API v2 with OAuth 1.0a

## Setup

See [SETUP.md](SETUP.md) for installation and configuration instructions.

## License

MIT
