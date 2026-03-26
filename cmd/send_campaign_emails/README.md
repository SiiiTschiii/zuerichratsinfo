# Send Campaign Emails

Send personalized campaign emails to Gemeinderat members with Bluesky accounts, announcing zuerichratsinfo on Bluesky.

## Prerequisites

- Go 1.21+
- Gmail account with [2-Step Verification](https://myaccount.google.com/security) enabled
- [Gmail App Password](https://myaccount.google.com/apppasswords) (create one named e.g. "zuerichratsinfo")

## Workflow

### Step 1: Verify the recipient list

```bash
go run cmd/send_campaign_emails/main.go
```

This prints a table with all recipients and their parameters:

```
#  Name              Email                       Gender     Salutation  Bluesky URL                          Source
1  Amstad Micha      micha.amstad@hotmail.com    männlich   Lieber      bsky.app/profile/michaamstad...       api
2  Alice Kohli       alice.kohli@sp6.ch          weiblich   Liebe       bsky.app/profile/aliwankoh...         override
...
```

Check that:

- All expected recipients are listed
- Gender and Salutation (Liebe/Lieber) are correct
- Bluesky URLs are correct
- Email addresses look right
- Source shows `api` (from Gemeinderat API) or `override` (from `data/email_overrides.yaml`)

### Step 2: Preview all rendered emails

```bash
# To stdout
go run cmd/send_campaign_emails/main.go --preview

# To a file for easier review
go run cmd/send_campaign_emails/main.go --preview --output data/emails_preview.md
```

Read through the fully rendered emails to verify the personalized text is correct.

### Step 3: Test send to yourself

```bash
export GMAIL_ADDRESS=you@gmail.com
export GMAIL_APP_PASSWORD=your-16-char-app-password

go run cmd/send_campaign_emails/main.go --test you@gmail.com
```

This sends **all emails to your address** instead of the real recipients. Check your inbox to verify subject, body, and formatting.

### Step 4: Send for real

```bash
export GMAIL_ADDRESS=you@gmail.com
export GMAIL_APP_PASSWORD=your-16-char-app-password

go run cmd/send_campaign_emails/main.go --send
```

Emails are sent with a 2-second delay between each to avoid rate limits.

## Email Overrides

Contacts not found in the Gemeinderat API (recently elected, or those with no email in the API) can be added manually in `data/email_overrides.yaml`:

```yaml
overrides:
  - name: Alice Kohli
    email: alice.kohli@sp6.ch
    gender: weiblich
```

The `name` must match exactly the name in `data/contacts.yaml`. Overrides take priority over API data.
