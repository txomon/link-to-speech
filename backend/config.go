package main

import (
	"log"
	"os"
	"strconv"
	"strings"
)

type Config struct {
	TelegramBotToken string
	OpenAIAPIKey     string
	AllowedUserIDs   []int64
	DefaultChatID    int64
	TTSVoice         string
	TTSModel         string
	ServerSecret     string
	ListenAddr       string
}

func LoadConfig() *Config {
	cfg := &Config{
		TelegramBotToken: os.Getenv("TELEGRAM_BOT_TOKEN"),
		OpenAIAPIKey:     os.Getenv("OPENAI_API_KEY"),
		TTSVoice:         envDefault("TTS_VOICE", "alloy"),
		TTSModel:         envDefault("TTS_MODEL", "tts-1"),
		ServerSecret:     os.Getenv("SERVER_SECRET"),
		ListenAddr:       envDefault("LISTEN_ADDR", ":8080"),
	}

	if cfg.TelegramBotToken == "" {
		log.Fatal("TELEGRAM_BOT_TOKEN is required")
	}
	if cfg.OpenAIAPIKey == "" {
		log.Fatal("OPENAI_API_KEY is required")
	}

	if ids := os.Getenv("ALLOWED_USER_IDS"); ids != "" {
		for _, s := range strings.Split(ids, ",") {
			s = strings.TrimSpace(s)
			if id, err := strconv.ParseInt(s, 10, 64); err == nil {
				cfg.AllowedUserIDs = append(cfg.AllowedUserIDs, id)
			}
		}
	}

	if chatID := os.Getenv("DEFAULT_CHAT_ID"); chatID != "" {
		if id, err := strconv.ParseInt(chatID, 10, 64); err == nil {
			cfg.DefaultChatID = id
		}
	}

	return cfg
}

func envDefault(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
