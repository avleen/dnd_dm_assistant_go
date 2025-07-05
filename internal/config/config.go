package config

import (
	"fmt"
	"log"
	"os"
	"regexp"
	"strconv"
	"strings"

	"github.com/joho/godotenv"
)

// Config holds all configuration for the bot
type Config struct {
	DiscordBotToken   string
	DMUserID          string
	DNDVoiceChannelID string
	CommandPrefix     string
	Debug             bool
}

const (
	// Discord snowflake IDs are 17-19 digit numbers
	discordIDPattern = `^\d{17,19}$`
)

// Load loads configuration from environment variables
func Load() (*Config, error) {
	// Try to load .env file if it exists
	envFile := ".env"
	if _, err := os.Stat(envFile); err == nil {
		if err := godotenv.Load(envFile); err != nil {
			log.Printf("Warning: Error loading .env file: %v", err)
		} else {
			log.Printf("Loaded environment variables from %s", envFile)
		}
	} else {
		log.Println("No .env file found - using system environment variables")
	}

	// Required environment variables
	requiredVars := []string{
		"DISCORD_BOT_TOKEN",
		"DM_USER_ID",
		"DND_VOICE_CHANNEL_ID",
	}

	var missingVars []string
	for _, varName := range requiredVars {
		if os.Getenv(varName) == "" {
			missingVars = append(missingVars, varName)
		}
	}

	if len(missingVars) > 0 {
		return nil, fmt.Errorf("missing required environment variables: %v", missingVars)
	}

	// Parse debug flag
	debug := false
	if debugStr := os.Getenv("DEBUG"); debugStr != "" {
		if parsed, err := strconv.ParseBool(debugStr); err == nil {
			debug = parsed
		}
	}

	config := &Config{
		DiscordBotToken:   os.Getenv("DISCORD_BOT_TOKEN"),
		DMUserID:          os.Getenv("DM_USER_ID"),
		DNDVoiceChannelID: os.Getenv("DND_VOICE_CHANNEL_ID"),
		CommandPrefix:     getEnvWithDefault("COMMAND_PREFIX", "!dnd"),
		Debug:             debug,
	}

	// Validate configuration
	if err := config.validate(); err != nil {
		return nil, fmt.Errorf("configuration validation failed: %w", err)
	}

	return config, nil
}

// validate validates the configuration values
func (c *Config) validate() error {
	// Validate Discord bot token format
	if !strings.HasPrefix(c.DiscordBotToken, "Bot ") &&
		!regexp.MustCompile(`^[A-Za-z0-9_-]+\.[A-Za-z0-9_-]+\.[A-Za-z0-9_-]+$`).MatchString(c.DiscordBotToken) {
		return fmt.Errorf("invalid Discord bot token format")
	}

	// Validate Discord IDs (snowflakes)
	discordIDRegex := regexp.MustCompile(discordIDPattern)

	if !discordIDRegex.MatchString(c.DMUserID) {
		return fmt.Errorf("invalid DM user ID format: must be a Discord snowflake (17-19 digits)")
	}

	if !discordIDRegex.MatchString(c.DNDVoiceChannelID) {
		return fmt.Errorf("invalid D&D voice channel ID format: must be a Discord snowflake (17-19 digits)")
	}

	// Validate command prefix
	if len(c.CommandPrefix) == 0 {
		return fmt.Errorf("command prefix cannot be empty")
	}

	return nil
}

// getEnvWithDefault returns environment variable value or default if not set
func getEnvWithDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
