# Changelog

All notable changes to the D&D DM Assistant Bot project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [0.1.0] - 2024-01-XX

At least 75% of this release was coded with Claude.

### Added
- **Complete Discord Bot Implementation**
  - Real-time voice channel monitoring and auto-join functionality
  - Command system with `!dnd` prefix support
  - Comprehensive error handling and graceful shutdown
  - Discord WebSocket connection management with automatic reconnection

- **Advanced Audio Processing Pipeline**
  - Per-speaker (SSRC) audio stream tracking and processing
  - Opus packet decoding and OGG container generation
  - Intelligent silence detection with 2-second threshold
  - Automatic cleanup of temporary audio files

- **Google Cloud Speech-to-Text Integration**
  - Batch speech recognition using REST API
  - Support for both v2 and v1p1beta1 APIs
  - Automatic audio format detection and configuration
  - Retry logic and error handling for API failures

- **Anthropic Claude AI Integration**
  - Persistent conversation history with JSON file storage
  - Intelligent context management with configurable message limits (default: 200)
  - Automatic transcription flushing every 10 seconds
  - Direct message delivery to DM user only

- **Robust Configuration Management**
  - Environment variable support with `.env` file loading
  - Comprehensive validation for Discord IDs and API keys
  - Required vs optional configuration handling
  - Clear error messages for configuration issues

- **Background Processing**
  - Auto-flush goroutine for timely Claude responses
  - Graceful shutdown handling for all background processes
  - Thread-safe conversation management

- **Command System**
  - `!dnd help` - Display help and current bot status
  - `!dnd status` - Show configuration and connection information
  - `!dnd flush` - Manually trigger transcription flush to Claude
  - `!dnd clear` - Clear conversation history (admin command)

- **Comprehensive Documentation**
  - Detailed README.md with features, setup, and usage instructions
  - Step-by-step SETUP.md guide for complete bot configuration
  - Technical documentation (TECHNICAL.md) for developers
  - Troubleshooting guide (TROUBLESHOOTING.md) for common issues
  - This changelog for tracking project evolution

### Technical Implementation Details

- **Architecture**: Modular design with clear separation of concerns
- **Language**: Go 1.24.3+ with modern concurrency patterns
- **Dependencies**: 
  - `github.com/bwmarrin/discordgo` for Discord API integration
  - `cloud.google.com/go/speech` for Google Speech-to-Text
  - `github.com/pion/webrtc/v3` for WebRTC audio processing
  - `github.com/joho/godotenv` for environment configuration

- **Security**: 
  - API keys managed via environment variables
  - No hardcoded credentials or sensitive information
  - Secure Discord token validation
  - Local-only conversation storage

- **Performance**:
  - Efficient memory management with streaming audio processing
  - Minimal CPU overhead with optimized audio pipelines
  - Background processing to prevent blocking main operations
  - Automatic cleanup of temporary resources

### Configuration

Required environment variables:
- `DISCORD_BOT_TOKEN` - Discord bot authentication token
- `DM_USER_ID` - Discord user ID of the Dungeon Master
- `DND_VOICE_CHANNEL_ID` - Voice channel ID for D&D sessions

Optional environment variables:
- `GOOGLE_APPLICATION_CREDENTIALS` - Path to Google Cloud service account key
- `GOOGLE_PROJECT_ID` - Google Cloud project ID for Speech-to-Text
- `ANTHROPIC_API_KEY` - Anthropic Claude API key for AI responses
- `CONVERSATION_FILE` - Path for conversation history storage (default: `dnd_conversation.json`)
- `MAX_CONVERSATION_MSGS` - Maximum conversation messages to retain (default: 200)
- `COMMAND_PREFIX` - Bot command prefix (default: `!dnd`)
- `DEBUG` - Enable debug logging (default: false)

### Known Limitations

- **Single Campaign Support**: Currently supports one conversation context
- **English Only**: Speech recognition configured for English (US)
- **Local Storage**: Conversation history stored locally (not distributed)
- **Single Server**: Designed for single Discord server deployment

### Target Use Cases

- **New Dungeon Masters**: Real-time assistance with D&D 5e rules and procedures
- **Experienced DMs**: AI support for complex rule interactions and edge cases
- **Remote D&D Groups**: Enhanced experience for voice-only gaming sessions
- **Learning Tool**: Helps players and DMs learn D&D 5e mechanics

### System Requirements

- **Operating System**: Windows, macOS, or Linux
- **Go Runtime**: Version 1.24.3 or higher
- **Network**: Internet access for Discord, Google Cloud, and Anthropic APIs
- **Storage**: Minimal disk space for conversation history and temporary audio files
- **CPU**: Adequate processing power for real-time audio processing

### Getting Started

1. Follow the comprehensive setup guide in [SETUP.md](SETUP.md)
2. Configure all required environment variables
3. Build and run the application: `go run main.go`
4. Invite bot to Discord server with appropriate permissions
5. Join configured voice channel to trigger bot auto-join
6. Begin your D&D session with AI assistance!

### Future Roadmap

_Claude made this up, but I kept it for ideas that I might implement one day_

Planned enhancements for future releases:
- Multi-campaign support with separate conversation contexts
- Web dashboard for real-time monitoring and configuration
- Custom AI prompt templates for different campaign styles
- Support for additional languages and regional dialects
- Integration with D&D Beyond for character and campaign data
- Voice synthesis for text-to-speech AI responses
- Advanced audio processing with noise reduction
- Database backend to replace JSON file storage

---

## Development History

This project evolved from a simple Discord bot concept to a sophisticated AI-powered DM assistant through several key development phases:

### Phase 1: Core Discord Integration
- Basic bot framework and Discord API connection
- Voice channel monitoring and auto-join functionality
- Command system implementation

### Phase 2: Audio Processing Pipeline
- Real-time audio capture from Discord voice channels
- Opus packet processing and OGG file generation
- Per-speaker audio stream management

### Phase 3: Speech Recognition Integration
- Google Cloud Speech-to-Text API integration
- Batch recognition workflow implementation
- Error handling and retry logic

### Phase 4: AI Integration
- Anthropic Claude API integration
- Conversation history management
- Automatic response generation and delivery

### Phase 5: Production Readiness
- Comprehensive error handling and logging
- Background processing and auto-flush functionality
- Configuration management and validation
- Complete documentation suite

### Phase 6: Documentation and Polish
- Comprehensive README and setup guides
- Technical documentation for developers
- Troubleshooting guides and common solutions
- Performance optimization and testing