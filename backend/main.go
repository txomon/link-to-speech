package main

import (
	"encoding/json"
	"log"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"syscall"
)

func main() {
	cfg := LoadConfig()

	if _, err := exec.LookPath("ffmpeg"); err != nil {
		log.Println("WARNING: ffmpeg not found — audio concatenation for long articles will fail")
	}

	bot, err := NewBot(cfg)
	if err != nil {
		log.Fatalf("Failed to create bot: %v", err)
	}

	// HTTP API for the Firefox addon
	mux := http.NewServeMux()
	mux.HandleFunc("/api/tts", addonHandler(bot, cfg))

	go func() {
		log.Printf("HTTP server listening on %s", cfg.ListenAddr)
		if err := http.ListenAndServe(cfg.ListenAddr, mux); err != nil {
			log.Fatalf("HTTP server: %v", err)
		}
	}()

	go bot.Start()

	log.Println("Bot is running. Press Ctrl+C to stop.")
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
	<-sig

	log.Println("Shutting down...")
	bot.Stop()
}

type addonRequest struct {
	Title  string `json:"title"`
	Text   string `json:"text"`
	URL    string `json:"url"`
	ChatID int64  `json:"chat_id,omitempty"`
}

func addonHandler(bot *Bot, cfg *Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		if cfg.ServerSecret != "" {
			if r.Header.Get("Authorization") != "Bearer "+cfg.ServerSecret {
				http.Error(w, "Unauthorized", http.StatusUnauthorized)
				return
			}
		}

		var req addonRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Invalid JSON", http.StatusBadRequest)
			return
		}
		if req.Text == "" {
			http.Error(w, "text is required", http.StatusBadRequest)
			return
		}

		chatID := req.ChatID
		if chatID == 0 {
			chatID = cfg.DefaultChatID
		}
		if chatID == 0 {
			http.Error(w, "No chat_id provided and no DEFAULT_CHAT_ID configured", http.StatusBadRequest)
			return
		}

		go bot.ProcessArticle(chatID, req.Title, req.Text, req.URL)

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]string{"status": "processing"})
	}
}
