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

## 5. Post to X

When you're ready to post the latest vote to X:

```bash
./run.sh
```

## What It Does

The bot automatically:

1. Fetches the latest council vote (Abstimmung) from Zurich's PARIS API
2. Formats it into a post with:
   - Vote date and result (accepted/rejected)
   - Vote description
   - Vote statistics (yes, no, abstentions, absent)
   - Shortened link to the official vote details
3. Posts it to X as @zuerichratsinfo

## Troubleshooting

- **"Missing X API credentials" error**: Make sure your `.env` file exists and contains all four credentials
- **Post too long**: The bot automatically shortens URLs using is.gd to save characters
- **API rate limits**: X API has rate limits. If you hit them, wait and try again later
