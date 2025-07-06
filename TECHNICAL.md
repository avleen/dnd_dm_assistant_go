# Technical Documentation

This document provides detailed technical information about the D&D DM Assistant Bot implementation.

## Architecture Overview

The bot follows a modular architecture with clear separation of concerns:

```
┌──────────────────────────────────────────────────────────────────┐
│                           Main Process                           │
├──────────────────────────────────────────────────────────────────┤
│                         Discord Bot                              │
│  ┌─────────────────┐  ┌─────────────────┐  ┌─────────────────┐   │
│  │ Event Handlers  │  │ Command Router  │  │ Voice Manager   │   │
│  └─────────────────┘  └─────────────────┘  └─────────────────┘   │
├──────────────────────────────────────────────────────────────────┤
│                      Audio Processing                            │
│  ┌─────────────────┐  ┌─────────────────┐  ┌─────────────────┐   │
│  │ Opus Decoder    │  │ SSRC Tracker    │  │ Silence Detect  │   │
│  └─────────────────┘  └─────────────────┘  └─────────────────┘   │
├──────────────────────────────────────────────────────────────────┤
│                      External Services                           │
│  ┌─────────────────┐  ┌─────────────────┐                        │
│  │ Google Speech   │  │ Anthropic API   │                        │
│  └─────────────────┘  └─────────────────┘                        │
└──────────────────────────────────────────────────────────────────┘
```

## Core Components

### 1. Main Application (`main.go`)

**Purpose**: Application entry point and lifecycle management

**Key Functions**:
- Configuration loading and validation
- Bot initialization and startup
- Graceful shutdown handling
- Signal management (SIGINT, SIGTERM)

**Dependencies**:
- `internal/config` - Configuration management
- `internal/bot` - Bot implementation

### 2. Configuration Management (`internal/config/config.go`)

**Purpose**: Centralized configuration loading and validation

**Key Features**:
- Environment variable loading with `.env` file support
- Required vs optional configuration handling
- Discord ID validation (snowflake format)
- Type conversion with defaults

**Configuration Structure**:
```go
type Config struct {
    // Discord
    DiscordBotToken   string
    DMUserID          string
    DNDVoiceChannelID string
    CommandPrefix     string
    Debug             bool
    
    // Google Cloud
    GoogleProjectID string
    GoogleCredsPath string
    
    // Anthropic Claude
    AnthropicAPIKey     string
    ConversationFile    string
    MaxConversationMsgs int
}
```

**Validation Rules**:
- Discord IDs must be 17-19 digit snowflakes
- Command prefix cannot be empty
- File paths are validated for existence when specified

### 3. Discord Bot (`internal/bot/bot.go`)

**Purpose**: Discord API integration and event handling

**Key Components**:
- **Session Management**: Discord WebSocket connection
- **Event Handlers**: Voice state, message, ready events
- **Command Processing**: Prefix-based command routing
- **Voice Channel Management**: Auto-join/leave logic

**Event Flow**:
```
Discord Event → Event Handler → Business Logic → Response
```

**Command System**:
- `!dnd help` - Display help and status
- `!dnd status` - Show configuration and connection info
- `!dnd flush` - Manually trigger transcription flush
- `!dnd clear` - Clear conversation history (admin only)

**Voice State Management**:
1. Monitor voice state updates for configured DM user
2. Auto-join when DM joins configured voice channel
3. Auto-leave when DM leaves or channel becomes empty
4. Handle reconnection scenarios

### 4. Audio Processing (`internal/audio/processor.go`)

**Purpose**: Real-time audio capture, processing, and transcription

**Audio Pipeline**:
```
Discord Opus → Opus Decode → OGG Container → Silence Detection → Transcription
```

**Key Features**:

#### SSRC (Synchronization Source) Tracking
- Each Discord user has a unique SSRC identifier
- Separate OGG files created for each SSRC
- Allows for per-speaker transcription accuracy

#### Silence Detection
- Monitors for silence packets
- Triggers transcription after 2 seconds of silence
- Prevents partial word transcription

#### OGG File Management
```go
type SSRCTracker struct {
    SSRC      uint32
    OGGFile   *os.File
    Encoder   *oggEncoder
    LastAudio time.Time
    Buffer    [][]byte
}
```

#### Transcription Flow
1. Audio buffered until silence detected
2. OGG file finalized and closed
3. File sent to Google Speech-to-Text API
4. Result passed to transcription callback
5. Temporary file cleaned up

### 5. Google Speech-to-Text Integration

**API Configuration**:
- Uses Cloud Speech-to-Text v2 and v1p1beta1 APIs
- REST API for batch recognition (not streaming)
- Automatic audio format detection

**Recognition Settings**:
```go
recognitionConfig := &speechpb.RecognitionConfig{
    Encoding:        speechpb.RecognitionConfig_OGG_OPUS,
    SampleRateHertz: 48000,
    LanguageCode:    "en-US",
    MaxAlternatives: 1,
    EnableAutomaticPunctuation: true,
}
```

**Error Handling**:
- Retry logic for temporary failures
- Graceful degradation when service unavailable
- Detailed error logging for debugging

### 6. Claude AI Integration (`internal/claude/`)

#### Service Management (`service.go`)
**Purpose**: Anthropic API communication and response handling

**Key Features**:
- HTTP client with timeout and retry logic
- Message formatting for Claude API
- Response parsing and error handling
- Token limit management

#### Conversation Management (`conversation.go`)
**Purpose**: Persistent conversation history and context management

**Key Components**:

**Conversation Structure**:
```go
type Conversation struct {
    Messages    []Message `json:"messages"`
    LastUpdated time.Time `json:"last_updated"`
    SessionID   string    `json:"session_id"`
}

type Message struct {
    Role      string    `json:"role"`      // "user" or "assistant"
    Content   string    `json:"content"`   // Message text
    Timestamp time.Time `json:"timestamp"` // When message was created
}
```

**Auto-Flush Mechanism**:
- Background goroutine runs every 10 seconds
- Checks for pending transcriptions
- Automatically sends accumulated transcriptions to Claude
- Handles API failures gracefully

**Message Management**:
- Maintains up to 200 messages (configurable)
- Implements FIFO cleanup when limit exceeded
- Preserves system prompt and recent context
- Disk persistence with JSON serialization

**System Prompt**:
The bot uses a system prompt that:
- Establishes Claude as a D&D 5e expert
- Encourages proactive but selective responses
- Provides guidance on when to respond vs stay silent
- Emphasizes helpfulness without being intrusive

### 7. Background Processing

**Auto-Flush Goroutine**:
```go
func (b *Bot) startAutoFlush() {
    go func() {
        ticker := time.NewTicker(10 * time.Second)
        defer ticker.Stop()
        
        for {
            select {
            case <-ticker.C:
                b.handleAutoFlush()
            case <-b.stopChan:
                return
            }
        }
    }()
}
```

**Benefits**:
- Ensures timely delivery of transcriptions to Claude
- Prevents loss of context during long conversations
- Reduces latency for AI responses
- Handles connection issues gracefully

## Data Flow

### 1. Audio Processing Flow
```
Voice Channel Audio → Opus Packets → Per-SSRC Processing → 
Silence Detection → Transcription → Claude Analysis → DM Response
```

### 2. Command Processing Flow
```
Discord Message → Command Parse → Permission Check → 
Business Logic → Response Generation → Discord Reply
```

### 3. Conversation Management Flow
```
Transcription → Add to Conversation → Check Limits → 
Persist to Disk → Send to Claude → Process Response → Send DM
```

## Performance Considerations

### Memory Management
- OGG files are streamed to disk, not held in memory
- Conversation history is loaded on demand
- Audio buffers are sized appropriately for latency vs memory trade-offs

### CPU Usage
- Opus decoding is efficient but will still cause a small amount of CPU usage
- Audio processing runs in separate goroutines
- Silence detection uses simple energy-based algorithms

### Network Optimization
- Batch transcription reduces API calls
- HTTP keep-alive for Claude API connections
- Graceful handling of network interruptions

### Storage
- Audio files are temporary and cleaned up immediately
- Conversation history grows slowly (text only)
- JSON persistence is efficient for conversation size

## Security Considerations

### API Key Management
- Environment variable isolation
- No hardcoded credentials
- Support for external secret management

### Network Security
- HTTPS/TLS for all external API calls
- Discord WebSocket security handled by library
- No sensitive data in logs (when DEBUG=false)

### Data Privacy
- Audio files are ephemeral (deleted after transcription)
- Conversation history stored locally only
- No data transmitted beyond configured APIs

### Access Control
- Discord permissions enforced
- Admin-only commands implemented
- DM-only response delivery

## Error Handling Strategy

### Recovery Mechanisms
1. **Connection Failures**: Automatic reconnection with backoff
2. **API Failures**: Graceful degradation and retry logic
3. **Audio Issues**: Continue processing other streams
4. **File System Issues**: Temporary directory fallback

### Logging Strategy
- Structured logging with levels (DEBUG, INFO, WARN, ERROR)
- Detailed context for debugging
- No sensitive information in logs
- Configurable log levels

### Graceful Degradation
- Bot continues functioning if transcription fails
- Manual commands work even if auto-flush fails
- Voice processing continues if Claude API is down

## Testing Strategy

### Unit Testing
- Configuration validation
- Audio processing components
- Conversation management
- API client functionality

### Integration Testing
- Discord API integration
- Google Speech API integration
- Anthropic API integration
- End-to-end audio flow

### Manual Testing
- Voice channel joining/leaving
- Command processing
- Real conversation scenarios
- Error condition handling

## Deployment Considerations

### Environment Requirements
- Go 1.24.3+ runtime
- Network access to Discord, Google Cloud, and Anthropic APIs
- Writable directory for conversation history
- Adequate CPU for real-time audio processing

### Configuration Management
- Environment variables for secrets
- File-based configuration for static settings
- Validation at startup
- Clear error messages for misconfigurations

### Monitoring
- Health check endpoints (future enhancement)
- Metrics collection (future enhancement)
- Log aggregation support
- Status command for runtime diagnostics

### Scaling Considerations
- Single-server deployment model
- Stateful conversation history (not horizontally scalable)
- API rate limiting awareness
- Memory usage monitoring recommended

## Future Enhancements

### Potential Improvements
1. **Web Dashboard**: Real-time monitoring and configuration
2. **Multiple Campaigns**: Support for multiple conversation contexts
3. **Custom Prompts**: Per-campaign AI prompt customization
4. **Advanced Audio**: Noise reduction and audio enhancement
5. **Speech Recognition**: On-device processing options
6. **Database Backend**: Replace JSON file storage
7. **Metrics**: Prometheus/Grafana integration
8. **Multi-Language**: Support for non-English campaigns

### API Enhancements
1. **Streaming Speech**: Real-time transcription
2. **Claude Function Calling**: Structured responses
3. **Discord Slash Commands**: Modern command interface
4. **Voice Synthesis**: Text-to-speech responses

## Development Guidelines

### Code Style
- Follow Go conventions and gofmt
- Comprehensive error handling
- Clear function and variable naming
- Adequate comments for complex logic

### Dependencies
- Minimize external dependencies
- Use stable, well-maintained libraries
- Regular dependency updates
- Security vulnerability monitoring

### Git Workflow
- Feature branches for new development
- Code review for all changes
- Comprehensive commit messages
- Tag releases with semantic versioning

This technical documentation should be updated as the system evolves and new features are added.
