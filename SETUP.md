# Setup Guide

## Prerequisites

- Go 1.21 or later
- X API credentials (OAuth 1.0a User Context)

## 1. Get X API Credentials

Go to https://developer.x.com/en/portal/dashboard

1. **Create a Project and App** (if you haven't already)
   - Give it a name like "Zurich Ratsinfo Bot"
2. **Set App Permissions**
   - Go to your app settings
   - Under "User authentication settings", click "Set up"
   - Enable "OAuth 1.0a"
   - Set permissions to "Read and Write"
   - Add a callback URL (can be http://localhost:3000 for testing)
   - Add a website URL (can be your GitHub repo)
3. **Get Your Credentials**
   - Go to "Keys and tokens" tab
   - Copy these 4 values:
     - API Key (Consumer Key)
     - API Key Secret (Consumer Secret)
     - Access Token
     - Access Token Secret

## 2. Install

```bash
git clone https://github.com/SiiiTschiii/zuerichratsinfo.git
cd zuerichratsinfo
```

## 3. Configure the App

```bash
# Copy the example environment file
cp .env.example .env

# Edit .env and add your credentials. Platform variables are prefixed with the
# channel key — the set of accounts they belong to.
ZURICH_X_API_KEY=your_api_key_here
ZURICH_X_API_SECRET=your_api_secret_here
ZURICH_X_ACCESS_TOKEN=your_access_token_here
ZURICH_X_ACCESS_SECRET=your_access_secret_here
```

**Important**: Never commit your `.env` file to git! It's already in `.gitignore`.

## 4. Preview Posts Locally

Nothing here posts anywhere or needs credentials — every command below prints to
your terminal or writes JPEGs to disk. This is the loop for reviewing how posts
read before they go anywhere near an account.

### Post text, from live data

`generate_vote_post` fetches the most recent votes and renders them exactly as
the bot would. It ignores the vote log, so already-posted votes still show up.

```bash
# Latest Gemeinderat vote group, all three platforms
go run ./cmd/generate_vote_post

# Latest Kantonsrat vote group
go run ./cmd/generate_vote_post -jurisdiction zurich-canton

# One platform, three groups
go run ./cmd/generate_vote_post -platform x -n 3
```

Rendering both bodies side by side is how you check a reader could not mistake
one for the other — they post to the same accounts:

```bash
go run ./cmd/generate_vote_post -jurisdiction zurich-city   -platform x -n 1
go run ./cmd/generate_vote_post -jurisdiction zurich-canton -platform x -n 1
```

Flags: `-jurisdiction` (`zurich-city`, `zurich-canton`), `-platform`
(`x`, `bluesky`, `instagram`; default all), `-n` groups to show, `-fetch` how
many individual votes to pull from the API.

### Images, from fixtures

`generate_vote_image` writes the Instagram carousel JPEGs. It runs off fixtures
rather than the live API, so it is fast, offline and reproducible.

```bash
# Every fixture, including the Kantonsrat one
go run ./cmd/generate_vote_image -out out/images

# One fixture, with the Instagram caption printed alongside
go run ./cmd/generate_vote_image -fixture kantonsrat-vote -platform instagram -out out/images
```

Open `out/images/` to compare. Each card carries a full-width band naming the
chamber in that body's colour, over a background keyed on jurisdiction plus
business number.

Fixture names come from `pkg/voteposting/testfixtures` — `single-vote-angenommen`,
`multi-vote-group`, `auswahl-vote`, `kantonsrat-vote`, and others; an unknown
name prints the full list.

### What the real bot would do right now

`check_unposted` is the dry-run mirror of `main.go`: it reads the actual vote
logs, applies each jurisdiction's age guard, and prints what would be posted —
including how the shared per-run budget is spent across both bodies. It
previews every jurisdiction, enabled or not.

```bash
go run ./cmd/check_unposted
go run ./cmd/check_unposted -platform x -jurisdiction zurich-canton -max-posts 10
```

This needs the vote logs on disk. They live on the `state-log` branch, not in
`main`, and are gitignored here:

```bash
git fetch origin state-log
mkdir -p data/zurich-city
for p in x bluesky instagram; do
  git show "FETCH_HEAD:data/zurich-city/posted_votes_$p.json" > "data/zurich-city/posted_votes_$p.json"
done
```

### Comparing the two data sources

Stadt Zürich is served by both PARIS and OpenParlData, which is what makes the
OpenParlData adapter checkable at all — the Kantonsrat has no second source to
verify against.

```bash
go run ./cmd/compare_sources -n 40
```

It exits non-zero if any totals, decisions or titles disagree.

## Development

### Testing

```bash
go test ./...
```

### E2E Testing with Test Accounts

E2E tests post real content to test accounts on X and Bluesky. They verify the full `Format → Post → API call` chain.

**Setup (one-time):**

1. Create test accounts on X and Bluesky (use obscure names, set to private/protected)
2. Fill in `.env.test` with the test account credentials
3. Edit `data/contacts_test.yaml` — replace placeholder handles with your test account handles

**Post fixtures to test accounts:**

```bash
source .env.test

# Post a single fixture to one platform
go run cmd/post_fixture/main.go --fixture=single-vote-angenommen --platform=x
go run cmd/post_fixture/main.go --fixture=multi-vote-group --platform=bluesky

# Post all fixtures to all platforms
go run cmd/post_fixture/main.go --fixture=all

# Use real contacts (tags real accounts — use with care)
go run cmd/post_fixture/main.go --contacts=data/zurich-city/contacts.yaml --fixture=vote-with-mentions

# Cleanup all test posts (will delete all posts made by the test accounts)
go run cmd/cleanup_posts/main.go
go run cmd/cleanup_posts/main.go --platform=x
go run cmd/cleanup_posts/main.go --platform=bluesky
```

**Test with live votes (fetches recent votes from the Zurich API):**

```bash
source .env.test
JURISDICTION_ZURICH_CITY_ENABLED=true SKIP_VOTE_LOG=true MAX_VOTES_TO_CHECK=5 go run main.go
```

No jurisdiction posts unless its `JURISDICTION_<KEY>_ENABLED` variable is `true`, so the switch has to be set here too.

**Regression workflow** (after formatting or posting changes):

1. `go test ./...` — automated unit tests, including the golden snapshot below
2. `source .env.test && go run cmd/post_fixture/main.go --fixture=all` — manual fixture verification
3. `source .env.test && JURISDICTION_ZURICH_CITY_ENABLED=true SKIP_VOTE_LOG=true MAX_VOTES_TO_CHECK=5 go run main.go` — manual live vote verification

### Golden Snapshot

`pkg/voteposting/golden` renders every fixture through all three platforms and
compares the result against `testdata/golden.txt`, verbatim. Generated images
are checked separately by their properties, because JPEG bytes differ between
CPU architectures and a byte hash would pass locally and fail in CI.

A refactor that is meant to preserve behaviour must leave this file untouched;
that is what makes the claim checkable rather than asserted. When a change is
*meant* to alter output, regenerate and read the diff before committing it:

```bash
go test ./pkg/voteposting/golden -update
git diff pkg/voteposting/golden/testdata/golden.txt
```

### Linting

Install and run [golangci-lint](https://golangci-lint.run/):

```bash
brew install golangci-lint  # macOS
golangci-lint run           # check all files
golangci-lint run --new     # only check unstaged changes
```

**VS Code**: Install the [Go extension](https://marketplace.visualstudio.com/items?itemName=golang.go) - settings are already configured in `.vscode/settings.json`.

### CI Locally

Run workflows locally with [act](https://github.com/nektos/act):

```bash
brew install act
act -W .github/workflows/go-ci.yml
```
