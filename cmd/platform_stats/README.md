# Platform Stats Command

This command analyzes the [contacts.yaml](../../data/contacts.yaml) file and updates the "Supported Platforms" table in the main README with current platform statistics.

## Usage

```bash
go run ./cmd/platform_stats/main.go
```

This will automatically update the platform counts in [README.md](../../README.md)'s "Supported Platforms" table.

## Output

The command updates:

- The Gemeinderäte column for each platform (LinkedIn, Facebook, Instagram, X, Bluesky, TikTok)
- The total contacts count in the footer note
- Shows a confirmation message with the updated statistics
