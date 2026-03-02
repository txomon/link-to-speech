# Backend

Go server providing a Telegram bot and an HTTP API for the Firefox addon.

## Configuration

Copy `.env.example` to `.env` and fill in the values:

| Variable | Required | Description |
|----------|----------|-------------|
| `TELEGRAM_BOT_TOKEN` | yes | Bot token from @BotFather |
| `OPENAI_API_KEY` | yes | OpenAI API key |
| `ALLOWED_USER_IDS` | no | Comma-separated Telegram user IDs (empty = allow all) |
| `DEFAULT_CHAT_ID` | no | Default Telegram chat ID for addon requests |
| `TTS_VOICE` | no | OpenAI voice: alloy, echo, fable, onyx, nova, shimmer (default: alloy) |
| `TTS_MODEL` | no | OpenAI TTS model: tts-1 or tts-1-hd (default: tts-1) |
| `SERVER_SECRET` | no | Shared secret for authenticating addon HTTP requests |
| `LISTEN_ADDR` | no | HTTP listen address (default: :8080) |

## Run with Docker

```bash
docker build -t link-to-speech .
docker run --env-file .env link-to-speech
```

## Run locally

Requires Go 1.22+ and ffmpeg.

```bash
go build -o link-to-speech .
export TELEGRAM_BOT_TOKEN=... OPENAI_API_KEY=...
./link-to-speech
```

## HTTP API

The addon sends extracted article text to:

```
POST /api/tts
Authorization: Bearer <SERVER_SECRET>
Content-Type: application/json

{"title": "...", "text": "...", "url": "...", "chat_id": 123456}
```

The `chat_id` field is optional; falls back to `DEFAULT_CHAT_ID`.
