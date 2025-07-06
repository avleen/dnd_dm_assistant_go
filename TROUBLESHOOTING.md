# Troubleshooting Guide

This guide helps you diagnose and resolve common issues with the D&D DM Assistant Bot.

## Table of Contents

1. [Quick Diagnostics](#quick-diagnostics)
2. [Configuration Issues](#configuration-issues)
3. [Discord Connection Problems](#discord-connection-problems)
4. [Audio Processing Issues](#audio-processing-issues)
5. [Speech-to-Text Problems](#speech-to-text-problems)
6. [Claude AI Issues](#claude-ai-issues)
7. [Performance Problems](#performance-problems)
8. [Debug Mode](#debug-mode)
9. [Common Error Messages](#common-error-messages)
10. [Getting Help](#getting-help)

## Quick Diagnostics

Before diving into specific issues, run these quick checks:

### 1. Basic Status Check
```bash
# Run the status command in Discord
!dnd status
```

This should show:
- Bot connection status
- Voice channel information
- API service availability
- Configuration summary

### 2. Environment Validation
```bash
# Build and run with immediate exit to check config
go run main.go
# Look for configuration loading messages
```

### 3. Permissions Check
Ensure your Discord bot has these permissions in your server:
- ✅ Send Messages
- ✅ Connect
- ✅ Speak  
- ✅ Use Voice Activity
- ✅ Message Content Intent

## Configuration Issues

### Problem: "Failed to load configuration" Error

**Symptoms**:
- Bot fails to start
- Error message about missing configuration

**Solutions**:

1. **Check .env file exists**:
   ```bash
   # In your project directory
   ls -la .env
   ```

2. **Verify required variables**:
   ```bash
   # Check for required variables
   grep -E "DISCORD_BOT_TOKEN|DM_USER_ID|DND_VOICE_CHANNEL_ID" .env
   ```

3. **Validate Discord IDs**:
   - Discord IDs must be 17-19 digits
   - Right-click in Discord and "Copy ID" (with Developer Mode enabled)

### Problem: "Invalid Discord ID format" Error

**Symptoms**:
- Configuration validation fails
- Error about Discord snowflake format

**Solutions**:

1. **Enable Discord Developer Mode**:
   - User Settings → Advanced → Developer Mode

2. **Get correct IDs**:
   ```
   User ID: Right-click username → Copy User ID
   Channel ID: Right-click channel name → Copy Channel ID
   Server ID: Right-click server name → Copy Server ID
   ```

3. **Validate ID format**:
   ```bash
   # Discord IDs should look like this:
   DM_USER_ID=xxxxxxxxxxxxxxxxxx        # ✅ 18 digits
   DND_VOICE_CHANNEL_ID=xxxxxxxxxxxxxxxxxxx  # ✅ 19 digits
   ```

## Discord Connection Problems

### Problem: Bot Doesn't Respond to Commands

**Symptoms**:
- `!dnd help` produces no response
- Bot appears offline in Discord

**Debugging Steps**:

1. **Check bot token**:
   ```bash
   # Test with a simple ping
   curl -H "Authorization: Bot YOUR_BOT_TOKEN" \
        https://discord.com/api/v10/users/@me
   ```

2. **Verify bot is in server**:
   - Check server member list for your bot
   - Re-invite bot if missing

3. **Check bot permissions**:
   - Server Settings → Roles → Your Bot Role
   - Ensure "Send Messages" is enabled

4. **Test in different channels**:
   - Try commands in various channels
   - Check channel-specific permissions

### Problem: Bot Doesn't Join Voice Channel

**Symptoms**:
- DM joins voice channel but bot doesn't follow
- No audio processing occurs

**Debugging Steps**:

1. **Verify voice channel ID**:
   ```bash
   # Check your .env file
   grep DND_VOICE_CHANNEL_ID .env
   ```

2. **Check voice permissions**:
   - Bot needs "Connect" permission for voice channel
   - Check channel-specific overrides

3. **Verify DM user ID**:
   ```bash
   # Ensure this matches the person who should trigger bot joining
   grep DM_USER_ID .env
   ```

4. **Test voice state detection**:
   - Enable debug mode
   - Watch logs when joining/leaving voice channel

## Audio Processing Issues

### Problem: No Audio Transcription

**Symptoms**:
- Bot joins voice channel but no transcriptions appear
- Debug logs show no audio processing

**Debugging Steps**:

1. **Check microphone permissions**:
   - Ensure Discord has microphone access
   - Test that Discord can hear your voice

2. **Verify audio codec support**:
   ```bash
   # Look for Opus packet processing in debug logs
   DEBUG=true go run main.go
   ```

3. **Test with different speakers**:
   - Try with multiple people speaking
   - Check if issue is user-specific

4. **Check silence detection**:
   - Speak continuously for 5+ seconds
   - Pause for 3+ seconds to trigger transcription

### Problem: Partial or Missing Audio

**Symptoms**:
- Some speech transcribed, some missing
- Transcriptions cut off mid-sentence

**Solutions**:

1. **Increase silence detection threshold**:
   - Current default: 2 seconds
   - May need adjustment for speaking patterns

2. **Check for network issues**:
   - Audio packets may be dropping
   - Monitor network stability

3. **Test with clear speech**:
   - Ensure speakers speak clearly
   - Minimize background noise

## Speech-to-Text Problems

### Problem: "Google Cloud Authentication Failed"

**Symptoms**:
- Audio processing works but no transcriptions
- Authentication errors in logs

**Solutions**:

1. **Verify service account key**:
   ```bash
   # Check file exists and is readable
   ls -la "$(echo $GOOGLE_APPLICATION_CREDENTIALS)"
   ```

2. **Test authentication**:
   ```bash
   # Test with gcloud CLI
   gcloud auth activate-service-account --key-file="$GOOGLE_APPLICATION_CREDENTIALS"
   gcloud auth list
   ```

3. **Check API is enabled**:
   - Google Cloud Console → APIs & Services → Library
   - Search for "Cloud Speech-to-Text API"
   - Ensure it's enabled

4. **Verify project ID**:
   ```bash
   # Check project ID matches Google Cloud Console
   grep GOOGLE_PROJECT_ID .env
   ```

### Problem: Poor Transcription Quality

**Symptoms**:
- Words frequently wrong or missing
- Transcriptions don't match speech

**Solutions**:

1. **Improve audio quality**:
   - Use good quality microphones
   - Minimize background noise
   - Speak clearly and at consistent volume

2. **Check language settings**:
   - Currently hardcoded to "en-US"
   - May need adjustment for accents/dialects

3. **Audio format issues**:
   - Ensure OGG Opus encoding is working correctly
   - Check sample rate settings (48kHz)

## Claude AI Issues

### Problem: No AI Responses

**Symptoms**:
- Transcriptions work but no Claude responses
- No DMs received from bot

**Debugging Steps**:

1. **Verify API key**:
   ```bash
   # Test API key with curl
   curl -H "Authorization: Bearer $ANTHROPIC_API_KEY" \
        -H "Content-Type: application/json" \
        https://api.anthropic.com/v1/messages
   ```

2. **Check DM permissions**:
   - Ensure bot can send DMs to configured DM user
   - User Settings → Privacy & Safety → Allow DMs from server members

3. **Test manual flush**:
   ```bash
   # In Discord
   !dnd flush
   ```

4. **Check conversation file**:
   ```bash
   # Look for conversation file
   ls -la dnd_conversation.json
   cat dnd_conversation.json | jq .
   ```

### Problem: Claude Responses Too Frequent/Infrequent

**Symptoms**:
- Claude responds to everything or nothing
- Responses not contextually appropriate

**Solutions**:

1. **Review system prompt**:
   - Check `internal/claude/service.go` for system prompt
   - Adjust guidance on when to respond

2. **Adjust auto-flush timing**:
   - Current: 10 seconds
   - May need tuning based on conversation pace

3. **Manual conversation management**:
   ```bash
   # Clear conversation history if needed
   !dnd clear
   ```

## Debug Mode

Enable comprehensive debug logging:

```bash
# In .env file
DEBUG=true

# Or set environment variable
export DEBUG=true
go run main.go
```

### Debug Output Sections

1. **Configuration Loading**:
   ```
   Loaded environment variables from .env
   Discord Bot Token: ****** (masked)
   DM User ID: 212424081205624842
   Voice Channel ID: 444965902761328643
   ```

2. **Discord Connection**:
   ```
   Bot is ready! 1 guilds, 42 users
   Voice state update: User xxxxxxxxxxxxxxxxxx joined channel xxxxxxxxxxxxxxxxxx
   ```

3. **Audio Processing**:
   ```
   New SSRC detected: 1234567890
   Audio packet received: 960 samples
   Silence detected, triggering transcription
   ```

4. **Transcription**:
   ```
   Transcription result: "I cast fireball at the goblins"
   Sending transcription to Claude
   ```

5. **Claude Integration**:
   ```
   Claude response: "Remember to have the goblins roll for Dexterity saves..."
   Sending DM to user xxxxxxxxxxxxxxxxxx
   ```

## Common Error Messages

### "Failed to create speech service"
**Cause**: Google Cloud authentication or API access issues  
**Solution**: Check service account key and API enablement

### "Discord token invalid"
**Cause**: Incorrect or expired bot token  
**Solution**: Regenerate token in Discord Developer Portal

### "Voice connection failed"
**Cause**: Missing voice permissions or channel access  
**Solution**: Check bot permissions and channel settings

### "Anthropic API error: 401"
**Cause**: Invalid or missing Claude API key  
**Solution**: Verify API key in Anthropic Console

### "No pending transcriptions to flush"
**Cause**: No recent voice activity or transcription failures  
**Solution**: Ensure audio processing is working first

### "Failed to send DM"
**Cause**: User has DMs disabled or bot blocked  
**Solution**: Check user's Discord privacy settings

## Performance Monitoring

### Monitor Key Metrics

1. **CPU Usage**:
   ```bash
   top -p $(pgrep dnd_dm_assistant)
   ```

2. **Memory Usage**:
   ```bash
   ps -o pid,ppid,%mem,rss,vsz,comm -p $(pgrep dnd_dm_assistant)
   ```

3. **Network Activity**:
   ```bash
   netstat -p | grep dnd_dm_assistant
   ```

4. **File Descriptors**:
   ```bash
   lsof -p $(pgrep dnd_dm_assistant) | wc -l
   ```

### Log Analysis

```bash
# Search for errors
grep -i error logs/bot.log

# Monitor transcription success rate
grep "Transcription result" logs/bot.log | wc -l

# Check Claude response frequency
grep "Claude response" logs/bot.log | wc -l
```

## Getting Help

### Information to Collect

When seeking help, gather:

1. **System Information**:
   - Operating system and version
   - Go version (`go version`)
   - Bot version/commit hash

2. **Configuration** (remove sensitive values):
   ```bash
   # Sanitized .env contents
   cat .env | sed 's/=.*/=***masked***/'
   ```

3. **Error Logs**:
   - Full error messages
   - Stack traces if available
   - Context around the error

4. **Steps to Reproduce**:
   - What you were doing when the issue occurred
   - Whether it's consistent or intermittent
   - Any recent changes made

### Check These Resources

1. **Documentation**: Re-read relevant sections of README.md and SETUP.md
2. **Known Issues**: Check GitHub issues for similar problems
3. **API Status**: Check status pages for Discord, Google Cloud, and Anthropic
4. **Community**: Discord servers or forums for the technologies used

### Creating Bug Reports

Include:
- Clear description of expected vs actual behavior
- Steps to reproduce the issue
- System information and logs
- Configuration details (sanitized)
- Any error messages or stack traces

This troubleshooting guide should help resolve most common issues. Keep it updated as new issues are discovered and resolved.
