# Link to Speech

Convert web articles into audio and receive them as Telegram voice messages.

Two input modes:

- **Telegram bot** — Send a URL directly to the bot. It extracts the article server-side and converts it to speech.
- **Firefox addon** — Click the toolbar button on any page (even behind paywalls/anti-bot walls). The addon extracts text from the live browser DOM using Readability.js and sends it to the backend.

## Architecture

```
Browser (Firefox addon)          Telegram
  │ Readability.js extraction      │ User sends URL
  │                                │
  ▼                                ▼
┌─────────────────────────────────────┐
│          Go Backend                 │
│  - Article extraction (go-readab.) │
│  - Text cleaning & chunking        │
│  - OpenAI TTS (opus)               │
│  - ffmpeg concatenation             │
│  - Telegram Bot API                 │
└─────────────────────────────────────┘
                │
                ▼
         Telegram voice message
```

## Components

### `backend/` — Go server

Telegram bot + HTTP API. Handles article extraction, text-to-speech via OpenAI API, and audio delivery.

See [backend/README.md](backend/README.md) for setup instructions.

### `firefox-addon/` — Firefox WebExtension

Extracts article text from the current page using Mozilla's Readability.js. Works on pages behind anti-bot protections, paywalls, and login walls because it operates on the fully-rendered DOM in a real browser session.

See [firefox-addon/README.md](firefox-addon/README.md) for installation instructions.

## Quick Start

1. Create a Telegram bot via [@BotFather](https://t.me/botfather) and save the token
2. Get an [OpenAI API key](https://platform.openai.com/api-keys)
3. Run the backend:
   ```bash
   cd backend
   cp .env.example .env
   # Edit .env with your tokens
   docker build -t link-to-speech .
   docker run --env-file .env link-to-speech
   ```
4. Message your bot `/start` to get your user/chat IDs, then set `ALLOWED_USER_IDS` and `DEFAULT_CHAT_ID`

## Requirements

- Go 1.22+ (for building without Docker)
- ffmpeg (for concatenating audio chunks from long articles)
- OpenAI API key (for TTS)
- Telegram Bot token
