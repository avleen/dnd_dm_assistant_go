package bot

import (
	"fmt"
	"log"
	"strings"
	"time"

	"dnd_dm_assistant_go/internal/audio"
	"dnd_dm_assistant_go/internal/config"
	"dnd_dm_assistant_go/internal/speech"

	"github.com/bwmarrin/discordgo"
)

const (
	// Startup delay to allow Discord state to stabilize
	startupDelay = 2 * time.Second

	// Command names
	commandJoin   = "join"
	commandLeave  = "leave"
	commandStatus = "status"
	commandHelp   = "help"
)

// Bot represents the D&D DM Assistant Discord bot
type Bot struct {
	config         *config.Config
	session        *discordgo.Session
	audioProcessor *audio.Processor
	speechService  *speech.Service
}

// New creates a new Bot instance
func New(cfg *config.Config) (*Bot, error) {
	// Create Discord session
	session, err := discordgo.New("Bot " + cfg.DiscordBotToken)
	if err != nil {
		return nil, fmt.Errorf("failed to create Discord session: %w", err)
	}

	// Set intents
	session.Identify.Intents = discordgo.IntentsAll

	// Create speech service if Google Cloud credentials are available
	var speechService *speech.Service
	if cfg.GoogleProjectID != "" {
		log.Printf("🔧 Attempting to create speech service with project ID: %s", cfg.GoogleProjectID)

		// Check if credentials file exists if specified
		if cfg.GoogleCredsPath != "" {
			log.Printf("🔧 Using credentials file: %s", cfg.GoogleCredsPath)
		} else {
			log.Printf("🔧 Using default credentials (ADC/environment)")
		}

		speechService, err = speech.NewService(cfg.GoogleProjectID, cfg.Debug)
		if err != nil {
			log.Printf("❌ Warning: Failed to create speech service: %v", err)
			log.Printf("   📋 Troubleshooting steps:")
			log.Printf("   1. Ensure GOOGLE_PROJECT_ID is set to your GCP project ID")
			log.Printf("   2. Set up authentication:")
			log.Printf("      • Set GOOGLE_APPLICATION_CREDENTIALS to path of service account JSON file")
			log.Printf("      • OR run 'gcloud auth application-default login'")
			log.Printf("      • OR use GCE/Cloud Run default credentials")
			if cfg.GoogleCredsPath != "" {
				log.Printf("   3. Check that credentials file exists: %s", cfg.GoogleCredsPath)
			}
			log.Printf("   🔗 See: https://cloud.google.com/docs/authentication/getting-started")
			log.Printf("   ⚠️  The bot will continue without speech-to-text functionality.")
			speechService = nil
		} else {
			log.Printf("✅ Speech service created successfully")
		}
	} else {
		log.Printf("ℹ️  Google Project ID not configured - speech service disabled")
		log.Printf("   Set GOOGLE_PROJECT_ID environment variable to enable speech-to-text")
	}

	// Create audio processor
	audioProcessor := audio.New(cfg.Debug)

	bot := &Bot{
		config:         cfg,
		session:        session,
		audioProcessor: audioProcessor,
		speechService:  speechService,
	}

	// Set up event handlers
	bot.setupEventHandlers()

	return bot, nil
}

// Start starts the bot
func (b *Bot) Start() error {
	// Open connection to Discord
	if err := b.session.Open(); err != nil {
		return fmt.Errorf("failed to open Discord session: %w", err)
	}

	log.Printf("Bot connected as %s", b.session.State.User.Username)
	log.Printf("Monitoring for DM user ID: %s", b.config.DMUserID)
	log.Printf("Target D&D voice channel ID: %s", b.config.DNDVoiceChannelID)

	return nil
}

// Stop stops the bot gracefully
func (b *Bot) Stop() {
	log.Printf("Shutting down bot gracefully...")

	// Stop audio processing first
	if b.audioProcessor != nil {
		log.Printf("Stopping audio processing...")
		b.audioProcessor.StopProcessing()
	}

	// Close speech service
	if b.speechService != nil {
		log.Printf("Closing speech service...")
		b.speechService.Close()
	}

	// Disconnect from all voice channels
	if b.session != nil {
		log.Printf("Disconnecting from voice channels...")
		for _, vc := range b.session.VoiceConnections {
			log.Printf("Disconnecting from voice channel in guild %s", vc.GuildID)
			vc.Disconnect()
		}

		// Close the Discord session
		log.Printf("Closing Discord session...")
		err := b.session.Close()
		if err != nil {
			log.Printf("Error closing Discord session: %v", err)
		} else {
			log.Printf("Discord session closed successfully")
		}
	}

	log.Printf("Bot shutdown complete")
}

// setupEventHandlers sets up Discord event handlers
func (b *Bot) setupEventHandlers() {
	b.session.AddHandler(b.onReady)
	b.session.AddHandler(b.onVoiceStateUpdate)
	b.session.AddHandler(b.onMessageCreate)
}

// onReady handles the ready event
func (b *Bot) onReady(s *discordgo.Session, event *discordgo.Ready) {
	log.Printf("Bot is ready! Logged in as %s", event.User.Username)

	// Check if DM is already in the target voice channel with fresh data
	go b.checkDMInVoiceChannelAsync()
}

// onVoiceStateUpdate handles voice state update events
func (b *Bot) onVoiceStateUpdate(s *discordgo.Session, vsu *discordgo.VoiceStateUpdate) {
	// Check if this is the DM user
	if vsu.UserID != b.config.DMUserID {
		return
	}

	// Check if DM joined the target voice channel
	if vsu.ChannelID == b.config.DNDVoiceChannelID {
		log.Printf("DM joined the D&D voice channel, joining...")
		b.joinVoiceChannel(vsu.GuildID, vsu.ChannelID)
	} else if vsu.BeforeUpdate != nil && vsu.BeforeUpdate.ChannelID == b.config.DNDVoiceChannelID {
		log.Printf("DM left the D&D voice channel, leaving...")
		b.leaveVoiceChannel(vsu.GuildID)
	}
}

// onMessageCreate handles message create events
func (b *Bot) onMessageCreate(s *discordgo.Session, m *discordgo.MessageCreate) {
	// Ignore messages from the bot itself
	if m.Author.ID == s.State.User.ID {
		return
	}

	// Handle commands
	if strings.HasPrefix(m.Content, b.config.CommandPrefix) {
		b.handleCommand(s, m)
	}
}

// handleCommand handles bot commands
func (b *Bot) handleCommand(s *discordgo.Session, m *discordgo.MessageCreate) {
	content := strings.TrimPrefix(m.Content, b.config.CommandPrefix)
	content = strings.TrimSpace(content)

	args := strings.Fields(content)
	if len(args) == 0 {
		return
	}

	command := strings.ToLower(args[0])

	switch command {
	case commandJoin:
		b.handleJoinCommand(s, m)
	case commandLeave:
		b.handleLeaveCommand(s, m)
	case commandStatus:
		b.handleStatusCommand(s, m)
	case commandHelp:
		b.handleHelpCommand(s, m)
	}
}

// handleJoinCommand handles the join command
func (b *Bot) handleJoinCommand(s *discordgo.Session, m *discordgo.MessageCreate) {
	// Find the guild
	guild, err := s.State.Guild(m.GuildID)
	if err != nil {
		log.Printf("Error finding guild %s: %v", m.GuildID, err)
		s.ChannelMessageSend(m.ChannelID, "❌ Unable to access guild information.")
		return
	}

	// Find the user's voice channel
	for _, vs := range guild.VoiceStates {
		if vs.UserID == m.Author.ID {
			b.joinVoiceChannel(guild.ID, vs.ChannelID)
			s.ChannelMessageSend(m.ChannelID, "✅ Joined your voice channel!")
			return
		}
	}

	s.ChannelMessageSend(m.ChannelID, "❌ You need to be in a voice channel first!")
}

// handleLeaveCommand handles the leave command
func (b *Bot) handleLeaveCommand(s *discordgo.Session, m *discordgo.MessageCreate) {
	b.leaveVoiceChannel(m.GuildID)
	s.ChannelMessageSend(m.ChannelID, "✅ Left the voice channel.")
}

// handleStatusCommand handles the status command
func (b *Bot) handleStatusCommand(s *discordgo.Session, m *discordgo.MessageCreate) {
	status := "✅ Bot is running\n"
	status += fmt.Sprintf("📡 Monitoring DM User: <@%s>\n", b.config.DMUserID)
	status += fmt.Sprintf("🎯 Target Voice Channel: <#%s>\n", b.config.DNDVoiceChannelID)

	if b.audioProcessor.IsProcessing() {
		status += "🎤 Currently processing audio\n"
	} else {
		status += "⏸️ Not processing audio\n"
	}

	if b.speechService != nil {
		status += "🗣️ Speech-to-text service: ✅ Active"
	} else {
		status += "🗣️ Speech-to-text service: ❌ Disabled"
	}

	s.ChannelMessageSend(m.ChannelID, status)
}

// handleHelpCommand handles the help command
func (b *Bot) handleHelpCommand(s *discordgo.Session, m *discordgo.MessageCreate) {
	help := "**D&D DM Assistant Bot Commands**\n\n"
	help += fmt.Sprintf("`%s %s` - Join your current voice channel\n", b.config.CommandPrefix, commandJoin)
	help += fmt.Sprintf("`%s %s` - Leave the current voice channel\n", b.config.CommandPrefix, commandLeave)
	help += fmt.Sprintf("`%s %s` - Show bot status\n", b.config.CommandPrefix, commandStatus)
	help += fmt.Sprintf("`%s %s` - Show this help message\n", b.config.CommandPrefix, commandHelp)
	help += "\n**Automatic Features:**\n"
	help += fmt.Sprintf("- Bot automatically joins when <@%s> joins <#%s>\n", b.config.DMUserID, b.config.DNDVoiceChannelID)

	s.ChannelMessageSend(m.ChannelID, help)
}

// checkDMInVoiceChannelAsync checks if the DM is already in the target voice channel
func (b *Bot) checkDMInVoiceChannelAsync() {
	log.Printf("Checking if DM is already in the target voice channel...")

	// Wait for Discord state to stabilize after connection
	time.Sleep(startupDelay)

	// Check each guild the bot is in
	for _, guild := range b.session.State.Guilds {
		if b.config.Debug {
			log.Printf("Checking guild: %s (ID: %s)", guild.Name, guild.ID)
		}

		// Verify the target channel exists in this guild
		if !b.isTargetChannelInGuild(guild.ID) {
			continue
		}

		// Check if DM is in target voice channel
		if b.isDMInTargetChannel(guild) {
			log.Printf("DM is already in the target D&D voice channel! Auto-joining...")
			b.joinVoiceChannel(guild.ID, b.config.DNDVoiceChannelID)
			return
		}
	}

	log.Printf("DM is not currently in the target D&D channel")
	log.Printf("Bot will monitor for voice state changes and auto-join when DM joins the target channel")
}

// isTargetChannelInGuild checks if the target voice channel exists in the given guild
func (b *Bot) isTargetChannelInGuild(guildID string) bool {
	targetChannel, err := b.session.Channel(b.config.DNDVoiceChannelID)
	if err != nil {
		if b.config.Debug {
			log.Printf("Could not fetch target channel %s: %v", b.config.DNDVoiceChannelID, err)
		}
		return false
	}

	if targetChannel.GuildID != guildID {
		if b.config.Debug {
			log.Printf("Target channel is not in this guild, skipping")
		}
		return false
	}

	if b.config.Debug {
		log.Printf("Found target D&D voice channel: %s", targetChannel.Name)
	}
	return true
}

// isDMInTargetChannel checks if the DM is currently in the target voice channel
func (b *Bot) isDMInTargetChannel(guild *discordgo.Guild) bool {
	for _, vs := range guild.VoiceStates {
		if vs.UserID == b.config.DMUserID {
			if b.config.Debug {
				log.Printf("Found DM in voice channel: %s", vs.ChannelID)
			}
			return vs.ChannelID == b.config.DNDVoiceChannelID
		}
	}
	return false
}

// joinVoiceChannel joins a voice channel and starts audio processing
func (b *Bot) joinVoiceChannel(guildID, channelID string) {
	log.Printf("Attempting to join voice channel %s in guild %s", channelID, guildID)

	// Join the voice channel with listening enabled
	// Parameters: guildID, channelID, mute=false, deaf=false
	vc, err := b.session.ChannelVoiceJoin(guildID, channelID, false, false)
	if err != nil {
		log.Printf("Error joining voice channel: %v", err)
		return
	}

	log.Printf("Successfully joined voice channel (listening enabled)")
	if b.config.Debug {
		log.Printf("Voice connection details: Ready=%v, UserID=%s", vc.Ready, vc.UserID)
	}

	// Start audio processing
	if err := b.audioProcessor.StartProcessing(vc); err != nil {
		log.Printf("Error starting audio processing: %v", err)
		// Still consider the join successful even if audio processing fails
		return
	}

	log.Printf("Started audio processing")
}

// leaveVoiceChannel leaves the current voice channel in the specified guild
func (b *Bot) leaveVoiceChannel(guildID string) {
	log.Printf("Attempting to leave voice channel in guild %s", guildID)

	// Stop audio processing first
	b.audioProcessor.StopProcessing()

	// Find and disconnect from the voice channel in this guild
	for _, vc := range b.session.VoiceConnections {
		if vc.GuildID == guildID {
			if err := vc.Disconnect(); err != nil {
				log.Printf("Error disconnecting from voice channel: %v", err)
			} else {
				log.Printf("Successfully left voice channel")
			}
			return
		}
	}

	log.Printf("No voice connection found for guild %s", guildID)
}
