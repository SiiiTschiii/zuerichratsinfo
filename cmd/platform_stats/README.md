# Platform Stats Command

This command analyzes the [contacts.yaml](../../data/contacts.yaml) file and automatically updates the "Supported Platforms" table in the main README.

## Usage

```bash
go run ./cmd/platform_stats/main.go
```

This will automatically update platform counts in [README.md](../../README.md).

## Output

The command:

- Updates the Gemeinderäte column for each platform (LinkedIn, Facebook, Instagram, X, Bluesky, TikTok)
- Updates the total contacts count in the footer
- Reports the update status to stderr

**Note:** If the README table structure changes and no longer matches the expected pattern, the update will be silently skipped without causing an error.
