# D&D DM Assistant Bot (Go)

A real-time Discord bot that helps Dungeon Masters run D&D 5e games by listening to voice calls and providing proactive guidance.

## Features

- **Real-time Audio Processing**: Joins Discord voice calls and processes streaming audio
- **Intelligent Triggering**: Monitors for specific users joining designated voice channels
- **Configurable**: Environment-based configuration for Discord tokens, channels, and users
- **Lightweight**: Built in Go for better performance and audio processing capabilities

## Architecture

1. Discord bot connects and monitors voice state changes
2. When configured DM joins the designated D&D voice channel, bot automatically joins
3. Captures streaming audio from all participants
4. Audio processing pipeline ready for speech-to-text integration
5. Extensible framework for adding AI-powered D&D guidance

## Requirements

- Go 1.21+
- Discord Bot Token
- Discord Bot Permissions: Voice, Message Content Intent

## Environment Variables

Create a `.env` file with the following variables:

```bash
DISCORD_BOT_TOKEN=your_discord_bot_token_here
DM_USER_ID=your_discord_user_id_here
DND_VOICE_CHANNEL_ID=your_dnd_voice_channel_id_here
COMMAND_PREFIX=!dnd
DEBUG=true
```

## Setup

1. Install dependencies: `go mod tidy`
2. Configure environment variables in `.env`
3. Run: `go run main.go`

## Usage

The bot will automatically join the configured D&D voice channel when the specified DM user joins it.

## Target Audience

New Dungeon Masters who need real-time guidance with D&D 5e rules and procedures.
