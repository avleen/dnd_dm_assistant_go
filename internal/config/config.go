package config

import (
	"fmt"
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

// Config holds all configuration for the bot
type Config struct {
	DiscordBotToken     string
	DMUserID            string
	DNDVoiceChannelID   string
	CommandPrefix       string
	Debug               bool
}

// Load loads configuration from environment variables
func Load() (*Config, error) {
	// Try to load .env file if it exists
	envFile := ".env"
	if _, err := os.Stat(envFile); err == nil {
		if err := godotenv.Load(envFile); err != nil {
			fmt.Printf("Warning: Error loading .env file: %v\n", err)
		} else {
			fmt.Printf("Loaded environment variables from %s\n", envFile)
		}
	} else {
		fmt.Println("No .env file found - using system environment variables")
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

	return config, nil
}

// getEnvWithDefault returns environment variable value or default if not set
func getEnvWithDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
