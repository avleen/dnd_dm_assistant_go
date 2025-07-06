# D&D DM Assistant Bot

A Discord bot that provides real-time assistance to Dungeon Masters during D&D 5e sessions.
The bot listens to voice channel conversations, transcribes each speaker, maintains persistent
conversation history, and uses Anthropic's Claude AI to provide proactive DM guidance via direct messages.

From the start most of this code was written with Claude, with strict direction and review from a human.

## ✨ Features

### 🎙️ Advanced Audio Processing
- **Real-time Voice Capture**: Automatically joins Discord voice channels and processes streaming Opus audio
- **Per-Speaker Transcription**: Creates separate OGG files for each speaker (SSRC) for accurate transcription
- **Intelligent Silence Detection**: Buffers audio and triggers transcription after 2 seconds of silence
- **Google Cloud Speech-to-Text Integration**: Uses v1p1beta1 APIs for high-quality transcription

### 🤖 AI-Powered Assistance
- **Anthropic Claude Integration**: AI assistant can be automatically and manually prompted for D&D 5e guidance
- **Persistent Conversation History**: Maintains context across sessions
- **Proactive Responses**: Automatically analyzes conversations and provides (usually) helpful suggestions
- **Direct Message Delivery**: All AI responses are sent privately to the DM via Discord DMs

### ⚡ Smart Automation
- **Auto-Join Voice Channels**: Automatically joins when the configured DM joins the D&D voice channel
- **Background Transcription Flushing**: Sends accumulated transcriptions to Claude every 10 seconds
- **Context Management**: Maintains up to 200 conversation messages with intelligent cleanup
- **Robust Error Handling**: Graceful handling of API failures and network issues

### 🎮 Discord Commands
- `!dnd help` - Show available commands and bot status
- `!dnd ask <question>` - Ask a specific question
- `!dnd status` - Display current bot configuration and connection status
- `!dnd flush` - Manually flush pending transcriptions to Claude
- `!dnd clear` - Clear conversation history (admin only)

## 🏗️ Architecture

```
┌─────────────────┐    ┌──────────────────┐    ┌─────────────────┐
│   Discord Bot   │    │ Audio Processor  │    │ Speech-to-Text  │
│                 │───▶│                  │───▶│   (Google)      │
│ - Voice Events  │    │ - SSRC Tracking  │    │ - REST API      │
│ - Commands      │    │ - Silence Det.   │    │ - Batch Recog.  │
└─────────────────┘    └──────────────────┘    └─────────────────┘
         │                                               │
         │              ┌─────────────────┐              │
         │              │ Claude Service  │◀─────────────┘
         │              │                 │
         │              │ - Conv. History │
         └─────────────▶│ - Auto Flush    │
                        │ - DM Responses  │
                        └─────────────────┘
```

### Component Details

1. **Discord Bot (`internal/bot/bot.go`)**
   - Manages Discord connection and voice state monitoring
   - Handles commands and voice channel auto-join
   - Coordinates between audio processing and AI services

2. **Audio Processor (`internal/audio/processor.go`)**
   - Processes incoming Opus packets from Discord voice channels
   - Maintains separate OGG files for each speaker (SSRC)
   - Implements silence detection and transcription triggering

3. **Claude Service (`internal/claude/service.go` & `conversation.go`)**
   - Manages conversations with Anthropic's Claude AI
   - Persists conversation history to disk
   - Handles auto-flushing and response delivery

4. **Configuration (`internal/config/config.go`)**
   - Environment-based configuration with validation
   - Supports both `.env` files and system environment variables

## 📋 Requirements

- **Go 1.24.3+**
- **Discord Bot Token** with the following permissions:
  - Send Messages
  - Connect to Voice Channels
  - Use Voice Activity
  - Message Content Intent
- **Google Cloud Project** with Speech-to-Text API enabled
- **Anthropic Claude API Key** (recommended: Claude-3.5-Sonnet)

## ⚙️ Configuration

### Environment Variables

Create a `.env` file in the project root with the following configuration:

```bash
# Discord Bot Configuration
DISCORD_BOT_TOKEN=your_discord_bot_token_here
DISCORD_GUILD_ID=your_discord_server_id_here
DND_VOICE_CHANNEL_ID=your_dnd_voice_channel_id_here

# Google Cloud Speech-to-Text
GOOGLE_APPLICATION_CREDENTIALS=/path/to/your/service-account.json
GOOGLE_PROJECT_ID=your_google_cloud_project_id

# Anthropic Claude API
ANTHROPIC_API_KEY=sk-ant-api03-your_api_key_here
CONVERSATION_FILE=dnd_conversation.json
MAX_CONVERSATION_MSGS=200

# Bot Configuration
DM_USER_ID=your_discord_user_id_here
COMMAND_PREFIX=!dnd
DEBUG=true
```

### Required Variables

| Variable | Description | Example |
|----------|-------------|---------|
| `DISCORD_BOT_TOKEN` | Your Discord bot token | `MTxxxxx.Gxxxxx.xxxxxxx` |
| `DM_USER_ID` | Discord user ID of the DM | `947264959326450960` |
| `DND_VOICE_CHANNEL_ID` | Voice channel ID for D&D sessions | `978547069317958426` |

### Optional Variables

| Variable | Description | Default |
|----------|-------------|---------|
| `COMMAND_PREFIX` | Bot command prefix | `!dnd` |
| `CONVERSATION_FILE` | Conversation history file | `dnd_conversation.json` |
| `MAX_CONVERSATION_MSGS` | Max messages in history | `200` |
| `DEBUG` | Enable debug logging | `false` |

## 🚀 Setup & Installation

### 1. Clone and Install Dependencies

```bash
git clone <repository-url>
cd dnd_dm_assistant_go
go mod tidy
```

### 2. Configure Discord Bot

1. Go to [Discord Developer Portal](https://discord.com/developers/applications)
2. Create a new application and bot
3. Copy the bot token to your `.env` file
4. Enable the following bot permissions:
   - Send Messages
   - Connect
   - Speak
   - Use Voice Activity
5. Enable Message Content Intent in Bot settings
6. Invite bot to your Discord server with these permissions

### 3. Setup Google Cloud Speech-to-Text

1. Create a Google Cloud Project
2. Enable the Speech-to-Text API
3. Create a service account and download the JSON key file
4. Set `GOOGLE_APPLICATION_CREDENTIALS` to the path of your key file
5. Set `GOOGLE_PROJECT_ID` to your project ID

### 4. Setup Anthropic Claude API

1. Sign up for [Anthropic API access](https://console.anthropic.com/)
2. Create an API key
3. Add the key to your `.env` file as `ANTHROPIC_API_KEY`

### 5. Configure Discord IDs

#### Finding Discord User ID (DM_USER_ID)
1. Enable Developer Mode in Discord (User Settings → Advanced → Developer Mode)
2. Right-click on your username and select "Copy User ID"

#### Finding Voice Channel ID (DND_VOICE_CHANNEL_ID)
1. Right-click on your D&D voice channel and select "Copy Channel ID"

### 6. Build and Run

```bash
# Build the application
go build -o dnd_dm_assistant main.go

# Run the bot
./dnd_dm_assistant

# Or run directly with Go
go run main.go
```

## 📖 Usage

### Basic Operation

1. **Start the Bot**: Run the application - the bot will connect to Discord
2. **Join Voice Channel**: When the configured DM joins the D&D voice channel, the bot automatically joins
3. **Automatic Transcription**: The bot begins processing audio from all participants
4. **AI Assistance**: Claude analyzes conversations and sends helpful suggestions via DM to the DM
5. **Manual Commands**: Use `!dnd` commands for additional control

### Available Commands

```
!dnd help     - Show this help message and bot status
!dnd status   - Show current bot configuration and connection status  
!dnd flush    - Manually flush pending transcriptions to Claude
!dnd clear    - Clear conversation history (admin command)
```

### How It Works

1. **Voice Detection**: Bot monitors when the DM joins the configured voice channel
2. **Audio Processing**: Captures and processes audio from all channel participants
3. **Transcription**: Converts speech to text using Google Cloud Speech-to-Text
4. **AI Analysis**: Sends transcriptions to Claude for D&D-specific analysis
5. **DM Assistance**: Claude's responses are delivered privately to the DM via Discord DMs

### Example Workflow

```
[DM joins voice channel] → [Bot auto-joins]
[Players discuss combat] → [Audio transcribed]
[Transcription sent to Claude every 10s] → [Claude analyzes for rule questions]
[Claude sends helpful response to DM privately]
```

## 🎯 Target Audience

- **New Dungeon Masters** who need real-time guidance with D&D 5e rules and procedures
- **Experienced DMs** who want AI assistance for complex rule interactions
- **D&D Groups** looking to enhance their gameplay experience with modern technology

## 🔧 Development

### Project Structure

```
dnd_dm_assistant_go/
├── main.go                 # Application entry point
├── go.mod                  # Go module dependencies
├── .env                    # Environment configuration
├── README.md               # This documentation
└── internal/
    ├── audio/
    │   └── processor.go    # Audio processing and transcription
    ├── bot/
    │   └── bot.go         # Discord bot implementation
    ├── claude/
    │   ├── service.go     # Claude API integration
    │   └── conversation.go # Conversation management
    └── config/
        └── config.go      # Configuration management
```

### Key Dependencies

- **github.com/bwmarrin/discordgo** - Discord API client
- **cloud.google.com/go/speech** - Google Cloud Speech-to-Text
- **github.com/pion/webrtc/v3** - WebRTC for audio processing
- **github.com/joho/godotenv** - Environment variable loading

## 🐛 Troubleshooting

### Common Issues

**Bot doesn't join voice channel:**
- Verify `DM_USER_ID` matches your Discord user ID
- Check `DND_VOICE_CHANNEL_ID` is correct
- Ensure bot has Connect permissions for the voice channel

**Transcription not working:**
- Verify Google Cloud credentials path is correct
- Check that Speech-to-Text API is enabled
- Ensure the service account has proper permissions

**Claude responses not received:**
- Verify `ANTHROPIC_API_KEY` is valid
- Check bot can send DMs to the configured DM user
- Ensure the DM has DMs enabled from server members

**Audio issues:**
- Verify Discord bot has Voice permissions
- Check that participants have microphones enabled
- Ensure voice channel isn't restricted

### Debug Mode

Enable debug logging by setting `DEBUG=true` in your `.env` file. This will provide detailed logs about:
- Voice state changes
- Audio processing events  
- Transcription results
- Claude API interactions

## 📄 License

[Include your license information here]

## 🤝 Contributing

[Include contribution guidelines here]

## 📞 Support

[Include support/contact information here]
