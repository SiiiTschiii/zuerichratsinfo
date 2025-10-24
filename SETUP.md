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

## 4. Run the App

```bash
./run.sh
```

## What It Does

1. Fetches the latest "Geschäft" (council business) from Zurich's council API
2. Formats it into a tweet like:

```
🏛️ Neues Geschäft im Gemeinderat Zürich

📋 2025/459: Motion
📅 01.10.2025 von Anjushka Früh (SP)

Strategie zur Einforderung eines angemessenen Anteils...
```

3. Posts it to X as @zuerichratsinfo

## Troubleshooting

- **403 Forbidden from X**: Check that your app has "Read and Write" permissions
- **401 Unauthorized from X**: Verify all 4 credentials are correct
- **403 from Zurich API**: This is normal from web browsers, but should work from the Go client
- **Environment variables not loading**: Make sure your `.env` file has no extra spaces or quotes around values

## 5. Set Up GitHub Actions (Optional)

To automate the bot with GitHub Actions:

1. **Add Repository Secrets**

   - Go to your GitHub repository
   - Navigate to Settings → Secrets and variables → Actions
   - Click "New repository secret" and add each of these:
     - `X_API_KEY` - Your API Key from X
     - `X_API_SECRET` - Your API Secret from X
     - `X_ACCESS_TOKEN` - Your Access Token from X
     - `X_ACCESS_SECRET` - Your Access Secret from X

2. **The Workflow is Already Set Up**

   - The workflow file is at `.github/workflows/bot.yml`
   - It runs hourly at minute 0 (e.g., 1:00, 2:00, 3:00, etc.)
   - You can also trigger it manually from the Actions tab

3. **Customize the Schedule**

   - Edit `.github/workflows/bot.yml`
   - Change the cron expression to your preferred schedule:
     - Hourly: `'0 * * * *'` (current setting)
     - Daily at 9 AM: `'0 9 * * *'`
     - Every 6 hours: `'0 */6 * * *'`
     - Twice daily (9 AM and 5 PM): `'0 9,17 * * *'`

4. **Monitor the Workflow**
   - Go to the "Actions" tab in your GitHub repository
   - You'll see the workflow runs and their logs

## Development

To test without posting to X, comment out the posting section in `main.go`:

```go
// err = postTweet(apiKey, apiSecret, accessToken, accessSecret, message)
// if err != nil {
//     log.Fatalf("Error posting tweet: %v", err)
// }
fmt.Println("(Posting disabled for testing)")
```
