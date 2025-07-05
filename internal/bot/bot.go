package bot

import (
	"fmt"
	"log"
	"strings"
	"time"

	"dnd_dm_assistant_go/internal/audio"
	"dnd_dm_assistant_go/internal/config"

	"github.com/bwmarrin/discordgo"
)

// Bot represents the D&D DM Assistant Discord bot
type Bot struct {
	config         *config.Config
	session        *discordgo.Session
	audioProcessor *audio.Processor
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

	// Create audio processor
	audioProcessor := audio.New(cfg.Debug)

	bot := &Bot{
		config:         cfg,
		session:        session,
		audioProcessor: audioProcessor,
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
	case "join":
		b.handleJoinCommand(s, m)
	case "leave":
		b.handleLeaveCommand(s, m)
	case "status":
		b.handleStatusCommand(s, m)
	case "help":
		b.handleHelpCommand(s, m)
	}
}

// handleJoinCommand handles the join command
func (b *Bot) handleJoinCommand(s *discordgo.Session, m *discordgo.MessageCreate) {
	// Find the guild
	guild, err := s.State.Guild(m.GuildID)
	if err != nil {
		log.Printf("Error finding guild: %v", err)
		return
	}

	// Find the user's voice channel
	for _, vs := range guild.VoiceStates {
		if vs.UserID == m.Author.ID {
			b.joinVoiceChannel(guild.ID, vs.ChannelID)
			return
		}
	}

	s.ChannelMessageSend(m.ChannelID, "You need to be in a voice channel first!")
}

// handleLeaveCommand handles the leave command
func (b *Bot) handleLeaveCommand(s *discordgo.Session, m *discordgo.MessageCreate) {
	b.leaveVoiceChannel(m.GuildID)
	s.ChannelMessageSend(m.ChannelID, "Left the voice channel.")
}

// handleStatusCommand handles the status command
func (b *Bot) handleStatusCommand(s *discordgo.Session, m *discordgo.MessageCreate) {
	status := "✅ Bot is running\n"
	status += fmt.Sprintf("📡 Monitoring DM User: <@%s>\n", b.config.DMUserID)
	status += fmt.Sprintf("🎯 Target Voice Channel: <#%s>\n", b.config.DNDVoiceChannelID)

	if b.audioProcessor.IsProcessing() {
		status += "🎤 Currently processing audio"
	} else {
		status += "⏸️ Not processing audio"
	}

	s.ChannelMessageSend(m.ChannelID, status)
}

// handleHelpCommand handles the help command
func (b *Bot) handleHelpCommand(s *discordgo.Session, m *discordgo.MessageCreate) {
	help := fmt.Sprintf("**D&D DM Assistant Bot Commands**\n\n")
	help += fmt.Sprintf("`%s join` - Join your current voice channel\n", b.config.CommandPrefix)
	help += fmt.Sprintf("`%s leave` - Leave the current voice channel\n", b.config.CommandPrefix)
	help += fmt.Sprintf("`%s status` - Show bot status\n", b.config.CommandPrefix)
	help += fmt.Sprintf("`%s help` - Show this help message\n", b.config.CommandPrefix)
	help += "\n**Automatic Features:**\n"
	help += fmt.Sprintf("- Bot automatically joins when <@%s> joins <#%s>\n", b.config.DMUserID, b.config.DNDVoiceChannelID)

	s.ChannelMessageSend(m.ChannelID, help)
}

// checkDMInVoiceChannelAsync checks if the DM is already in the target voice channel
// This function fetches fresh guild data to ensure accurate voice state information
func (b *Bot) checkDMInVoiceChannelAsync() {
	log.Printf("Checking if DM is already in the target voice channel...")

	// Wait a moment for Discord state to stabilize after connection
	time.Sleep(2 * time.Second)

	// Check each guild the bot is in
	for _, guild := range b.session.State.Guilds {
		log.Printf("Checking guild: %s (ID: %s)", guild.Name, guild.ID)

		// Check if the target voice channel exists in this guild
		targetChannel, err := b.session.Channel(b.config.DNDVoiceChannelID)
		if err != nil {
			log.Printf("Could not fetch target channel %s: %v", b.config.DNDVoiceChannelID, err)
			continue
		}

		// Make sure the channel is in this guild
		if targetChannel.GuildID != guild.ID {
			log.Printf("Target channel is not in guild %s, skipping", guild.Name)
			continue
		}

		log.Printf("Found target D&D voice channel: %s in guild %s", targetChannel.Name, guild.Name)

		// Method 1: Check the session state first (most reliable for cached data)
		log.Printf("Checking session state for voice states...")
		for _, vs := range guild.VoiceStates {
			if vs.UserID == b.config.DMUserID {
				log.Printf("Found DM in voice channel: %s (from session state)", vs.ChannelID)
				if vs.ChannelID == b.config.DNDVoiceChannelID {
					log.Printf("DM is already in the target D&D voice channel! Auto-joining...")
					b.joinVoiceChannel(guild.ID, vs.ChannelID)
					return
				} else {
					log.Printf("DM is in a different voice channel (ID: %s), not auto-joining", vs.ChannelID)
					return
				}
			}
		}

		// Method 2: If not found in session state, try to get fresh voice states from the channel itself
		log.Printf("DM not found in session state, checking channel members...")

		// Get the target channel and verify it exists
		_, err = b.session.Channel(b.config.DNDVoiceChannelID)
		if err != nil {
			log.Printf("Error fetching fresh channel data: %v", err)
			continue
		}

		// For voice channels, we need to iterate through guild members and check their voice state
		// Since the Guild() API doesn't populate voice states, we'll use a different approach
		log.Printf("Attempting to fetch guild members to check voice states...")

		// Try to get guild members (this might be limited by intents)
		members, err := b.session.GuildMembers(guild.ID, "", 1000)
		if err != nil {
			log.Printf("Could not fetch guild members (this might be due to missing intents): %v", err)
			log.Printf("Falling back to checking if user is in target channel using different method...")

			// Method 3: Try a more direct approach - attempt to get the specific user
			member, err := b.session.GuildMember(guild.ID, b.config.DMUserID)
			if err != nil {
				log.Printf("Could not fetch DM member from guild: %v", err)
				continue
			}

			log.Printf("Successfully fetched DM member: %s", member.User.Username)
			// Unfortunately, Member objects don't contain voice state either
			// We'll have to rely on the session state or voice state updates
			log.Printf("Member object doesn't contain voice state, will rely on voice state updates")
			continue
		}

		log.Printf("Successfully fetched %d members from guild", len(members))

		// Check each member for the DM
		for _, member := range members {
			if member.User.ID == b.config.DMUserID {
				log.Printf("Found DM member: %s", member.User.Username)
				// Note: Member objects don't contain voice state information
				// Voice states are separate and not included in member data
				break
			}
		}
	}

	log.Printf("DM is not currently in the target D&D channel or voice state not available in cache")
	log.Printf("Bot will monitor for voice state changes and auto-join when DM joins the target channel")
}

// joinVoiceChannel joins a voice channel and starts audio processing
func (b *Bot) joinVoiceChannel(guildID, channelID string) {
	// Join the voice channel
	vc, err := b.session.ChannelVoiceJoin(guildID, channelID, false, true)
	if err != nil {
		log.Printf("Error joining voice channel: %v", err)
		return
	}

	log.Printf("Successfully joined voice channel")

	// Start audio processing
	if err := b.audioProcessor.StartProcessing(vc); err != nil {
		log.Printf("Error starting audio processing: %v", err)
		return
	}

	log.Printf("Started audio processing")
}

// leaveVoiceChannel leaves the current voice channel
func (b *Bot) leaveVoiceChannel(guildID string) {
	// Stop audio processing
	b.audioProcessor.StopProcessing()

	// Leave voice channel
	for _, vc := range b.session.VoiceConnections {
		if vc.GuildID == guildID {
			vc.Disconnect()
			log.Printf("Left voice channel")
			return
		}
	}
}
