# Zurich Ratsinfo

A minimal civic tech project that fetches the latest vote from the Zurich City Council (Gemeinderat Zürich) and posts it to X (Twitter) as [@zuerichratsinfo](https://x.com/zuerichratsinfo).

## Features

- Fetches latest voting information from the [Zurich Council PARIS API](https://opendatazurich.github.io/paris-api/)
- Posts updates to X/Twitter automatically
- Simple, single-binary Go application

## Prerequisites

- Go 1.21 or later
- X API credentials (OAuth 1.0a User Context)

## Getting X API Credentials

1. Go to the [X Developer Portal](https://developer.x.com/en/portal/dashboard)
2. Create a new project and app (or use an existing one)
3. Navigate to your app's "Keys and tokens" tab
4. You'll need the following credentials:
   - **API Key** and **API Key Secret** (Consumer Keys)
   - **Access Token** and **Access Token Secret** (Authentication Tokens)
5. Make sure your app has **Read and Write** permissions
6. Copy all four credentials - you'll need them in the next step

**Note**: For posting tweets, X requires OAuth 1.0a User Context or OAuth 2.0 User Context. Bearer tokens (App-Only) cannot be used to post tweets.

## Setup

1. Clone this repository:

```bash
cd /Users/cgerber/code/zuerichratsinfo
```

2. Create a `.env` file based on `.env.example`:

```bash
cp .env.example .env
```

3. Edit `.env` and add your X API credentials:

```bash
X_API_KEY=your_api_key_here
X_API_SECRET=your_api_secret_here
X_ACCESS_TOKEN=your_access_token_here
X_ACCESS_SECRET=your_access_secret_here
```

## Running

You can run the application directly:

```bash
# Load environment variables and run
source <(grep -v '^#' .env | sed 's/^/export /')
go run .
```

Or build and run:

```bash
# Build the binary
go build -o zurichratsinfo

# Run with environment variables
source <(grep -v '^#' .env | sed 's/^/export /')
./zurichratsinfo
```

## How It Works

1. The program fetches the latest "Geschäft" (council business/matter) from `https://www.gemeinderat-zuerich.ch/api/`
2. This includes motions, written questions, and other council submissions
3. Formats the information into a tweet (max 280 characters) with:
   - GR number and type (Motion, Schriftliche Anfrage, etc.)
   - Date submitted
   - Author and party
   - Title of the submission
4. Posts the tweet to X using the @zuerichratsinfo account via X API v2 with OAuth 1.0a
5. Displays confirmation or error messages

## Project Structure

```
.
├── main.go           # Main application entry point
├── council_api.go    # Zurich Council PARIS API client (XML parsing)
├── twitter_api.go    # X API v2 client with OAuth 1.0a User Context
├── go.mod            # Go module definition
├── run.sh            # Helper script to load .env and run the program
├── .env.example      # Example environment variables
└── README.md         # This file
```

## Future Extensions

This is a minimal prototype. Planned extensions:

- Scheduled automatic posting (cron job or GitHub Actions)
- More detailed vote information
- Historical vote tracking
- Additional social media platforms
- Web dashboard

## API Documentation

- [PARIS API Documentation](https://opendatazurich.github.io/paris-api/)
- [X API v2 Documentation](https://docs.x.com/x-api/getting-started/about-x-api)
- [X API v2 Sample Code](https://github.com/xdevplatform/Twitter-API-v2-sample-code)

## License

MIT
