# Send Campaign Emails

Send personalized campaign emails to Gemeinderat members with an account on a given social platform, announcing zuerichratsinfo on that platform.

## Supported platforms

- `bluesky`
- `instagram`

Planned future additions: `facebook`, `tiktok`, `linkedin` (once those project accounts are active).

## Prerequisites

- Go 1.21+
- Gmail account with [2-Step Verification](https://myaccount.google.com/security) enabled
- [Gmail App Password](https://myaccount.google.com/apppasswords) (create one named e.g. "zuerichratsinfo")

## Workflow

### Step 1: Verify the recipient list

```bash
go run cmd/send_campaign_emails/main.go --platform instagram
```

This prints a table with all recipients and their parameters:

```
#  Name              Email                       Gender     Salutation  Instagram URL                        Source
1  Amstad Micha      micha.amstad@hotmail.com    männlich   Lieber      instagram.com/michaamstad...          api
2  Alice Kohli       alice.kohli@sp6.ch          weiblich   Liebe       instagram.com/aliwankoh...            override
...
```

Check that:

- All expected recipients are listed
- Gender and Salutation (Liebe/Lieber) are correct
- Platform URLs are correct
- Email addresses look right
- Source shows `api` (from Gemeinderat API) or `override` (from `data/email_overrides.yaml`)

### Step 2: Preview all rendered emails

```bash
# To stdout
go run cmd/send_campaign_emails/main.go --platform instagram --preview

# To a file for easier review
go run cmd/send_campaign_emails/main.go --platform instagram --preview --output data/emails_preview_instagram.md
```

Read through the fully rendered emails to verify the personalized text is correct.

### Step 3: Test send to yourself

```bash
export GMAIL_ADDRESS=you@gmail.com
export GMAIL_APP_PASSWORD=your-16-char-app-password

go run cmd/send_campaign_emails/main.go --platform instagram --test you@gmail.com
```

This sends **all emails to your address** instead of the real recipients. Check your inbox to verify subject, body, and formatting.

### Step 4: Send for real

```bash
export GMAIL_ADDRESS=you@gmail.com
export GMAIL_APP_PASSWORD=your-16-char-app-password

go run cmd/send_campaign_emails/main.go --platform instagram --send
```

Emails are sent with a 2-second delay between each to avoid rate limits.

Use `--platform bluesky` to run the same workflow for Bluesky contacts.

## Email Overrides

Contacts not found in the Gemeinderat API (recently elected, or those with no email in the API) can be added manually in `data/email_overrides.yaml`:

```yaml
overrides:
  - name: Alice Kohli
    email: alice.kohli@sp6.ch
    gender: weiblich
```

The `name` must match exactly the name in `data/contacts.yaml`. Overrides take priority over API data.
