package main

import (
	"context"
	"fmt"
	"log"
	"regexp"
	"strings"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

var linkRe = regexp.MustCompile(`https?://[^\s<>"]+`)

type Bot struct {
	api *tgbotapi.BotAPI
	cfg *Config
	tts *TTSClient
}

func NewBot(cfg *Config) (*Bot, error) {
	api, err := tgbotapi.NewBotAPI(cfg.TelegramBotToken)
	if err != nil {
		return nil, fmt.Errorf("creating bot: %w", err)
	}
	log.Printf("Authorized on account %s", api.Self.UserName)

	return &Bot{
		api: api,
		cfg: cfg,
		tts: NewTTSClient(cfg.OpenAIAPIKey, cfg.TTSModel, cfg.TTSVoice),
	}, nil
}

func (b *Bot) Start() {
	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60
	for update := range b.api.GetUpdatesChan(u) {
		if update.Message == nil {
			continue
		}
		go b.handleMessage(update.Message)
	}
}

func (b *Bot) Stop() {
	b.api.StopReceivingUpdates()
}

// --- message handling ---

func (b *Bot) handleMessage(msg *tgbotapi.Message) {
	if !b.isAllowed(msg.From.ID) {
		b.reply(msg, fmt.Sprintf("Unauthorized. Your user ID: %d", msg.From.ID))
		return
	}

	if msg.IsCommand() {
		b.handleCommand(msg)
		return
	}

	if url := extractURL(msg.Text); url != "" {
		b.handleURL(msg, url)
		return
	}

	b.reply(msg, "Send me a URL and I'll convert the article to audio.")
}

func (b *Bot) handleCommand(msg *tgbotapi.Message) {
	switch msg.Command() {
	case "start":
		b.reply(msg, fmt.Sprintf(
			"Welcome! Send me a URL and I'll convert the article to audio.\n\nYour chat ID: %d\nYour user ID: %d",
			msg.Chat.ID, msg.From.ID))
	case "help":
		b.reply(msg, "Send me a URL to convert an article to audio.\n\nI'll extract the text, generate speech with OpenAI TTS, and send you a voice message.")
	default:
		b.reply(msg, "Unknown command. Send /help for usage info.")
	}
}

func (b *Bot) handleURL(msg *tgbotapi.Message, url string) {
	statusID := b.reply(msg, "Extracting article...")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	article, err := ExtractArticle(ctx, url)
	if err != nil {
		b.edit(msg.Chat.ID, statusID, fmt.Sprintf("Failed to extract article: %s", err))
		return
	}

	clean := CleanText(article.Content)
	if clean == "" {
		b.edit(msg.Chat.ID, statusID, "No readable content found.")
		return
	}

	fullText := clean
	if article.Title != "" {
		fullText = article.Title + ".\n\n" + clean
	}

	chunks := ChunkText(fullText, maxChunkSize)
	b.edit(msg.Chat.ID, statusID,
		fmt.Sprintf("Generating audio... (%d chars, %d chunk(s))", len(fullText), len(chunks)))

	audio, err := b.tts.GenerateAudio(ctx, fullText)
	if err != nil {
		b.edit(msg.Chat.ID, statusID, fmt.Sprintf("TTS failed: %s", err))
		return
	}

	b.edit(msg.Chat.ID, statusID, "Sending audio...")
	b.sendVoice(msg, article.Title, url, audio)
	b.delete(msg.Chat.ID, statusID)
}

// ProcessArticle handles text submitted by the Firefox addon via the HTTP API.
func (b *Bot) ProcessArticle(chatID int64, title, text, url string) {
	if chatID == 0 {
		log.Println("ProcessArticle: no chat ID")
		return
	}

	statusID := b.send(chatID, fmt.Sprintf("Processing: %s", title))

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	clean := CleanText(text)
	if clean == "" {
		b.edit(chatID, statusID, "No readable content in the submitted text.")
		return
	}

	fullText := clean
	if title != "" {
		fullText = title + ".\n\n" + clean
	}

	chunks := ChunkText(fullText, maxChunkSize)
	b.edit(chatID, statusID,
		fmt.Sprintf("Generating audio... (%d chars, %d chunk(s))", len(fullText), len(chunks)))

	audio, err := b.tts.GenerateAudio(ctx, fullText)
	if err != nil {
		b.edit(chatID, statusID, fmt.Sprintf("TTS failed: %s", err))
		return
	}

	caption := title
	if url != "" {
		caption += "\n" + url
	}

	voice := tgbotapi.NewVoice(chatID, tgbotapi.FileBytes{Name: "article.ogg", Bytes: audio})
	voice.Caption = truncate(caption, 1024)
	if _, err := b.api.Send(voice); err != nil {
		b.edit(chatID, statusID, fmt.Sprintf("Failed to send audio: %s", err))
		return
	}

	b.delete(chatID, statusID)
}

// --- auth ---

func (b *Bot) isAllowed(userID int64) bool {
	if len(b.cfg.AllowedUserIDs) == 0 {
		return true
	}
	for _, id := range b.cfg.AllowedUserIDs {
		if id == userID {
			return true
		}
	}
	return false
}

// --- telegram helpers ---

func (b *Bot) reply(msg *tgbotapi.Message, text string) int {
	m := tgbotapi.NewMessage(msg.Chat.ID, text)
	m.ReplyToMessageID = msg.MessageID
	sent, err := b.api.Send(m)
	if err != nil {
		log.Printf("reply error: %v", err)
		return 0
	}
	return sent.MessageID
}

func (b *Bot) send(chatID int64, text string) int {
	sent, err := b.api.Send(tgbotapi.NewMessage(chatID, text))
	if err != nil {
		log.Printf("send error: %v", err)
		return 0
	}
	return sent.MessageID
}

func (b *Bot) edit(chatID int64, msgID int, text string) {
	if msgID == 0 {
		return
	}
	if _, err := b.api.Send(tgbotapi.NewEditMessageText(chatID, msgID, text)); err != nil {
		log.Printf("edit error: %v", err)
	}
}

func (b *Bot) delete(chatID int64, msgID int) {
	if msgID == 0 {
		return
	}
	if _, err := b.api.Request(tgbotapi.NewDeleteMessage(chatID, msgID)); err != nil {
		log.Printf("delete error: %v", err)
	}
}

func (b *Bot) sendVoice(msg *tgbotapi.Message, title, url string, audio []byte) {
	caption := title
	if url != "" {
		caption += "\n" + url
	}

	voice := tgbotapi.NewVoice(msg.Chat.ID, tgbotapi.FileBytes{Name: "article.ogg", Bytes: audio})
	voice.Caption = truncate(caption, 1024)
	voice.ReplyToMessageID = msg.MessageID

	if _, err := b.api.Send(voice); err != nil {
		log.Printf("sendVoice error: %v", err)
		b.reply(msg, fmt.Sprintf("Failed to send audio: %s", err))
	}
}

// --- util ---

func extractURL(text string) string {
	match := linkRe.FindString(text)
	return strings.TrimRight(match, ".,;:!?)")
}

func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max-3] + "..."
}
