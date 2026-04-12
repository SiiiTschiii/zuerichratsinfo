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

# Edit .env and add your credentials
X_API_KEY=your_api_key_here
X_API_SECRET=your_api_secret_here
X_ACCESS_TOKEN=your_access_token_here
X_ACCESS_SECRET=your_access_secret_here
```

**Important**: Never commit your `.env` file to git! It's already in `.gitignore`.

## 4. Test Without Posting

Preview what will be posted without actually posting to X:

```bash
# Preview the latest vote
go run cmd/generate_vote_post/main.go

# Preview the last 5 votes
go run cmd/generate_vote_post/main.go 5
```

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
go run cmd/post_fixture/main.go --contacts=data/contacts.yaml --fixture=vote-with-mentions

# Cleanup all test posts (will delete all posts made by the test accounts)
go run cmd/cleanup_posts/main.go
go run cmd/cleanup_posts/main.go --platform=x
go run cmd/cleanup_posts/main.go --platform=bluesky
```

**Test with live votes (fetches recent votes from the Zurich API):**

```bash
source .env.test
SKIP_VOTE_LOG=true MAX_VOTES_TO_CHECK=5 go run main.go
```

**Regression workflow** (after formatting or posting changes):

1. `go test ./...` — automated unit tests
2. `source .env.test && go run cmd/post_fixture/main.go --fixture=all` — manual fixture verification
3. `source .env.test && SKIP_VOTE_LOG=true MAX_VOTES_TO_CHECK=5 go run main.go` — manual live vote verification

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
