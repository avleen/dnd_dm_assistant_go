# Setup Guide

This guide will walk you through setting up the D&D DM Assistant Bot step by step.

## Prerequisites

Before you begin, ensure you have:

- [Go 1.24.3+](https://golang.org/dl/) installed
- A Discord account with server admin permissions
- Access to Google Cloud Platform
- An Anthropic API account

## Step 1: Discord Bot Setup

### 1.1 Create Discord Application

1. Go to the [Discord Developer Portal](https://discord.com/developers/applications)
2. Click "New Application"
3. Give your application a name (e.g., "D&D DM Assistant")
4. Click "Create"

### 1.2 Create Bot User

1. In your application, navigate to the "Bot" section
2. Click "Add Bot"
3. Under "Token", click "Copy" to copy your bot token
4. Save this token securely - you'll need it for the `.env` file

### 1.3 Configure Bot Permissions

1. In the "Bot" section, enable the following under "Privileged Gateway Intents":
   - ✅ **Message Content Intent** (required for commands)
   - ✅ **Server Members Intent** (recommended)

2. Under "Bot Permissions", ensure the following are enabled:
   - ✅ Send Messages
   - ✅ Connect
   - ✅ Speak
   - ✅ Use Voice Activity

### 1.4 Invite Bot to Server

1. Go to the "OAuth2" → "URL Generator" section
2. Under "Scopes", select:
   - ✅ `bot`
   - ✅ `applications.commands`

3. Under "Bot Permissions", select:
   - ✅ Send Messages
   - ✅ Connect
   - ✅ Speak
   - ✅ Use Voice Activity

4. Copy the generated URL and open it in your browser
5. Select your Discord server and authorize the bot

## Step 2: Google Cloud Speech-to-Text Setup

### 2.1 Create Google Cloud Project

1. Go to [Google Cloud Console](https://console.cloud.google.com/)
2. Click "Select a project" → "New Project"
3. Give your project a name and click "Create"
4. Note your Project ID for later use

### 2.2 Enable Speech-to-Text API

1. In the Google Cloud Console, navigate to "APIs & Services" → "Library"
2. Search for "Cloud Speech-to-Text API"
3. Click on it and click "Enable"

### 2.3 Create Service Account

1. Navigate to "IAM & Admin" → "Service Accounts"
2. Click "Create Service Account"
3. Enter a name (e.g., "dnd-bot-speech") and description
4. Click "Create and Continue"
5. Add the role "Cloud Speech Client" 
6. Click "Continue" and then "Done"

### 2.4 Generate Service Account Key

1. Click on your newly created service account
2. Go to the "Keys" tab
3. Click "Add Key" → "Create new key"
4. Select "JSON" format and click "Create"
5. Save the downloaded JSON file securely
6. Note the full path to this file for your `.env` configuration

## Step 3: Anthropic Claude API Setup

### 3.1 Create Anthropic Account

1. Go to [Anthropic Console](https://console.anthropic.com/)
2. Sign up for an account or log in
3. Complete any required verification steps

### 3.2 Generate API Key

1. In the Anthropic Console, navigate to "API Keys"
2. Click "Create Key"
3. Give your key a name (e.g., "D&D Bot")
4. Copy the generated API key
5. Store it securely for your `.env` file

## Step 4: Discord Configuration

### 4.1 Enable Developer Mode

1. In Discord, go to User Settings (gear icon)
2. Navigate to "Advanced"
3. Enable "Developer Mode"

### 4.2 Get Your User ID (DM_USER_ID)

1. Right-click on your username in Discord
2. Select "Copy User ID"
3. Save this ID for your `.env` file

### 4.3 Get Voice Channel ID

1. Navigate to your D&D voice channel
2. Right-click on the channel name
3. Select "Copy Channel ID"
4. Save this ID for your `.env` file

### 4.4 Get Server ID (Optional)

1. Right-click on your server name
2. Select "Copy Server ID"
3. Save this ID for your `.env` file

## Step 5: Project Setup

### 5.1 Clone or Download Project

```bash
git clone <repository-url>
cd dnd_dm_assistant_go
```

### 5.2 Install Dependencies

```bash
go mod tidy
```

### 5.3 Create Environment Configuration

Create a `.env` file in the project root:

```bash
# Discord Bot Configuration
DISCORD_BOT_TOKEN=your_bot_token_from_step_1.2
DISCORD_GUILD_ID=your_server_id_from_step_4.4
DND_VOICE_CHANNEL_ID=your_voice_channel_id_from_step_4.3

# Google Cloud Speech-to-Text
GOOGLE_APPLICATION_CREDENTIALS=C:\path\to\your\service-account.json
GOOGLE_PROJECT_ID=your_project_id_from_step_2.1

# Anthropic Claude API
ANTHROPIC_API_KEY=your_api_key_from_step_3.2
CONVERSATION_FILE=dnd_conversation.json
MAX_CONVERSATION_MSGS=200

# Bot Configuration
DM_USER_ID=your_user_id_from_step_4.2
COMMAND_PREFIX=!dnd
DEBUG=true
```

**Important**: Replace all placeholder values with your actual configuration values.

## Step 6: Testing the Setup

### 6.1 Build the Application

```bash
go build -o dnd_dm_assistant main.go
```

### 6.2 Run the Bot

```bash
./dnd_dm_assistant
```

You should see output similar to:
```
2024/01/XX XX:XX:XX Loaded environment variables from .env
2024/01/XX XX:XX:XX Bot is ready! 1 guilds, 1 users
D&D DM Assistant Bot is running. Press CTRL-C to exit.
```

### 6.3 Test Bot Connection

1. In Discord, type `!dnd status` in any channel where the bot has read permissions
2. The bot should respond with its current status

### 6.4 Test Voice Functionality

1. Join your configured D&D voice channel
2. The bot should automatically join the channel
3. Speak something - you should see transcription activity in the logs (if DEBUG=true)

## Step 7: Verification Checklist

Before using the bot in a real D&D session, verify:

- [ ] Bot responds to `!dnd help` command
- [ ] Bot automatically joins voice channel when DM joins
- [ ] Audio transcription is working (check debug logs)
- [ ] Claude API is responding (test with `!dnd flush`)
- [ ] Bot sends DMs to the configured DM user
- [ ] All error logs are resolved

## Troubleshooting

### Common Setup Issues

**"Failed to load configuration" error:**
- Check that your `.env` file exists in the project root
- Verify all required variables are set and not empty
- Ensure Discord IDs are 17-19 digit numbers

**"Invalid token" Discord error:**
- Verify your bot token is correct and hasn't been regenerated
- Ensure there are no extra spaces or characters
- Try regenerating the token in Discord Developer Portal

**Google Cloud authentication errors:**
- Verify the service account JSON file path is correct
- Ensure the file has proper read permissions
- Check that the Speech-to-Text API is enabled for your project

**Bot doesn't join voice channel:**
- Verify the voice channel ID is correct
- Check that the bot has Connect permissions for the channel
- Ensure your user ID matches the configured DM_USER_ID

**Claude API errors:**
- Verify your API key is valid and has sufficient credits
- Check for API rate limits or usage restrictions
- Ensure network connectivity to Anthropic's servers

### Debug Mode

For detailed troubleshooting, enable debug mode by setting `DEBUG=true` in your `.env` file. This will show:

- Configuration loading details
- Discord connection events
- Voice state changes
- Audio processing information
- Transcription results
- Claude API interactions

### Getting Help

If you encounter issues not covered here:

1. Check the console output for error messages
2. Enable debug mode for more detailed logs
3. Verify all setup steps were completed correctly
4. Check that all APIs and services are accessible from your network

## Security Notes

- Keep your `.env` file secure and never commit it to version control
- Store API keys and tokens securely
- Regularly rotate your API keys and bot tokens
- Use appropriate file permissions for credential files
- Consider using environment variables instead of `.env` files in production

## Next Steps

Once setup is complete:

1. Read the main [README.md](README.md) for usage instructions
2. Test the bot with a small group before using in actual sessions
3. Familiarize yourself with all available commands
4. Consider customizing the Claude system prompt for your campaign style
