# Send Campaign Emails

Send personalized campaign emails announcing zuerichratsinfo. Two kinds of campaign:

- **`--platform`** — to Gemeinderat members who have an account on a given social platform (emails pulled from the Zürich PARIS API + overrides). Message: "we're now on <platform>, you were tagged".
- **`--audience`** — general outreach to a whole audience (parties, cantonal/federal politicians). Recipients come from a curated local YAML file; message is the general intro + heads-up announcement.

Exactly one of `--platform` or `--audience` is required.

## Supported platforms (`--platform`)

- `bluesky`
- `instagram`

Planned future additions: `facebook`, `tiktok`, `linkedin` (once those project accounts are active).

## Supported audiences (`--audience`)

| Audience                    | Recipient file (local, gitignored)                  |
| --------------------------- | --------------------------------------------------- |
| `city-parties`              | `data/campaign_recipients/city_parties.yaml`        |
| `cantonal-national-parties` | `data/campaign_recipients/cantonal_national_parties.yaml` |
| `cantonal-zh`               | `data/campaign_recipients/cantonal_zh_politicians.yaml`   |
| `federal-zh`                | `data/campaign_recipients/federal_zh_politicians.yaml`    |

The workflow (verify → preview → test → send) below is identical for both kinds; just swap `--platform instagram` for e.g. `--audience city-parties`.

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

## Audience recipient files

For `--audience` campaigns, recipients are read from a curated YAML file (one per audience, see the table above). These files hold personal contact emails and are **gitignored** — they must not be committed. Only `data/campaign_recipients/recipients.example.yaml` (schema, no real emails) is committed.

Schema (see the example file for details):

```yaml
recipients:
  - name: SP Stadt Zürich       # org: the party name
    email: info@spzuerich.ch
    type: org                   # "person" (default) or "org"
    party: SP
  - name: Erika Musterfrau      # person
    email: erika@example.ch
    type: person
    gender: weiblich            # drives Liebe/Lieber salutation
    role: Kantonsrätin
    party: GLP
```

- `type: org` → greeting "Guten Tag" (uses *ihr*); `type: person` → "Liebe/Lieber <Name>" (uses *du*).
- Entries without an `email` are skipped with a warning.
- `--recipients <file>` overrides the default file for an audience (handy for testing against `recipients.example.yaml`).

### Sourcing the recipient lists

Both parliaments publish members' contact emails through their data services, so
most entries need no manual filling (verified 2026-07):

- **Federal (ZH):** roster via parlament.ch OData
  (`MemberCouncil`, `CantonAbbreviation eq 'ZH' and Active eq true`); each member's
  *published* email via the legacy REST API
  `ws-old.parlament.ch/councillors/<id>?format=json` (`contact.emailWork`).
  Don't guess `firstname.lastname@parl.ch` — several members publish a different
  address. A few publish none; leave `email` empty so the tool skips them.
- **Cantonal (ZH):** `kantonsrat.zh.ch/mitglieder/` ships the full roster —
  including published emails, party, and gender — in the page's Nuxt
  `_payload.json`. Regierungsrat members (`zh.ch/de/regierungsrat.html`) publish
  no personal emails; use the Direktionsassistenz/office address from each
  member's zh.ch page.
- **Parties:** official contact addresses from each party's website (some hide
  them behind Cloudflare `data-cfemail` obfuscation or block scraping — check
  the impressum page, or fill manually). Watch for shared secretariats: the
  same address can appear in both the city and the cantonal list.

Always review the list via `--preview` before any real send.
