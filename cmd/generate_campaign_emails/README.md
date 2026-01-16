# Generate Campaign Emails

This tool generates personalized campaign emails for Gemeinderat and Stadtrat members who have both:
1. An account on a specified social media platform
2. An email address in the Gemeinderat API

## Quick Start

```bash
# Generate emails for X (Twitter) contacts
go run cmd/generate_campaign_emails/main.go -platform x -output data/emails_x.md

# Generate emails for other platforms
go run cmd/generate_campaign_emails/main.go -platform instagram -output data/emails_instagram.md
go run cmd/generate_campaign_emails/main.go -platform facebook -output data/emails_facebook.md
go run cmd/generate_campaign_emails/main.go -platform linkedin -output data/emails_linkedin.md
go run cmd/generate_campaign_emails/main.go -platform bluesky -output data/emails_bluesky.md
```

The tool automatically fetches fresh contact data from the Gemeinderat API using the existing `zurichapi.Client`.

## Usage

## Flags

- `-platform`: Social media platform (x, instagram, facebook, linkedin, bluesky, tiktok) [default: "x"]
- `-contacts`: Path to contacts YAML file [default: "data/contacts.yaml"]
- `-output`: Output file path (if not specified, outputs to stdout)

## How It Works

1. Loads contacts from `data/contacts.yaml`
2. Filters for contacts with accounts on the specified platform
3. Fetches all contact details from the Gemeinderat API using `zurichapi.Client.FetchAllKontakte()`
4. Matches contacts by name and extracts email addresses and gender
5. Generates personalized emails with appropriate salutations

## Features

- ✅ Uses existing API infrastructure (`zurichapi.Client`)
- ✅ No manual curl commands or caching needed
- ✅ Supports all major social media platforms
- ✅ Gender-appropriate salutations (Liebe/Lieber)
- ✅ Role-appropriate greetings (Gemeinderat/Gemeinderätin, Stadtrat/Stadträtin)
- ✅ Personalized with recipient's platform URL
- ✅ Professional campaign email template
- ✅ Markdown formatted output ready to send

## Output Format

The tool generates a markdown file with personalized emails containing:
- Recipient name and email address
- Gender and role-appropriate salutation
- Project description
- Link to the recipient's social media account
- Call to action
- Professional closing with Wahlkampf wish
