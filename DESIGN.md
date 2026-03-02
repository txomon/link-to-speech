# Article-to-Audio Pipeline: Firefox Addon + Telegram Bot

## Problem Statement

The user wants to convert web articles into audio (text-to-speech) and receive the resulting audio file via Telegram. The key challenge is that many websites employ anti-bot and anti-crawler protections (Cloudflare challenges, JavaScript rendering walls, cookie consent gates, paywalls requiring login, etc.), which means server-side scraping tools fail to retrieve the actual article content. A solution that extracts text from a real, fully-rendered browser session is required.

## Proposed Solution: Hybrid Architecture

A two-component system:

1. **Firefox WebExtension (addon)** — Extracts article text from the currently loaded page in the user's browser, leveraging the fact that the user has already navigated past all anti-bot protections, logged in, accepted cookies, etc.
2. **Backend Server with Telegram Bot** — Receives the extracted text, calls a TTS API to generate audio, and sends the audio file to the user via Telegram.

---

## Component 1: Firefox WebExtension

### Purpose

Extract clean article text from the current browser tab and send it to the backend server.

### Key Technical Details

- **Cannot access Reader Mode directly.** Firefox's `about:reader?url=...` pages are privileged and block content script injection from WebExtensions.
- **Use Mozilla's Readability.js instead.** This is the same library that powers Firefox Reader Mode. It runs against the live DOM and returns clean article content. Repository: https://github.com/mozilla/readability
- **Operates on the fully-rendered DOM**, meaning all anti-bot protections, JS rendering, and authentication have already been resolved by the user's normal browsing session.

### Flow

1. User navigates to an article in Firefox (normal browsing, not Reader Mode).
2. User clicks the addon's browser action button (toolbar icon).
3. The addon injects `Readability.js` and a content script into the active tab.
4. The content script clones the document, runs Readability.js, and extracts `{ title, textContent, excerpt, byline, siteName }`.
5. The content script sends the extracted data to the background script via `browser.runtime.sendMessage()`.
6. The background script POSTs the data to the backend server.

### Addon Structure

```
firefox-addon/
├── manifest.json
├── background.js          # Listens for button click, injects scripts, sends to backend
├── content-script.js      # Runs Readability.js on the page DOM, sends result back
├── readability.js         # Mozilla Readability.js library (vendored from @mozilla/readability npm)
├── options.html           # Settings page for server URL, secret, chat ID
├── options.js             # Settings page logic (browser.storage.local)
└── icons/                 # TODO: needs icon-48.png and icon-96.png
```

### manifest.json Permissions Required

- `activeTab` — grants access to the current tab only when the user clicks the button. No need for `<all_urls>`.
- `storage` — for persisting settings (server URL, secret, chat ID).
- `notifications` — for success/failure feedback after sending.

### Content Script (Pseudocode)

```javascript
// content-script.js
// Readability requires a cloned document to avoid mutating the live page
const documentClone = document.cloneNode(true);
const article = new Readability(documentClone).parse();

if (article) {
  browser.runtime.sendMessage({
    type: "article_extracted",
    title: article.title,
    text: article.textContent,      // Plain text, suitable for TTS
    excerpt: article.excerpt,
    byline: article.byline,
    siteName: article.siteName,
    url: window.location.href
  });
} else {
  browser.runtime.sendMessage({
    type: "extraction_failed",
    url: window.location.href
  });
}
```

### Background Script (Pseudocode)

```javascript
// background.js
const SERVER_URL = "https://your-server.com/api/tts"; // Configurable

browser.browserAction.onClicked.addListener(async (tab) => {
  // Inject Readability.js first, then the content script
  await browser.tabs.executeScript(tab.id, { file: "readability.js" });
  await browser.tabs.executeScript(tab.id, { file: "content-script.js" });
});

browser.runtime.onMessage.addListener(async (msg) => {
  if (msg.type === "article_extracted") {
    try {
      const response = await fetch(SERVER_URL, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({
          title: msg.title,
          text: msg.text,
          url: msg.url
        })
      });
      // Optionally show a notification on success/failure
    } catch (error) {
      console.error("Failed to send to server:", error);
    }
  }
});
```

### UX Considerations

- Show a brief browser notification or badge on success ("Audio sent to Telegram!") or failure.
- Optionally, an options page where the user can configure: server URL, Telegram chat ID, TTS voice preferences.
- Consider adding a visual indicator (badge text or icon change) while processing.

---

## Component 2: Backend Server + Telegram Bot

### Purpose

Receive extracted article text, generate audio via a TTS API, and deliver the audio file to the user's Telegram chat.

### Tech Stack

- **Go**: Standard library `net/http` for the HTTP endpoint, `go-telegram-bot-api/v5` for Telegram integration, `go-shiori/go-readability` for server-side article extraction.

### Flow

1. Receive POST request from the Firefox addon with `{ title, text, url }`.
2. (Optional) Preprocess text: remove excessive whitespace, truncate if too long for TTS API limits, split into chunks if needed.
3. Call TTS API with the text.
4. Receive audio file (typically MP3 or OGG).
5. Send audio to the user's Telegram chat via the Telegram Bot API.

### API Endpoint

```
POST /api/tts
Authorization: Bearer <SERVER_SECRET>
Content-Type: application/json

{
  "title": "Article Title",
  "text": "Full article plain text...",
  "url": "https://example.com/article",
  "chat_id": 123456789           // optional, falls back to DEFAULT_CHAT_ID
}

Response: 200 OK { "status": "processing" }
```

Processing is asynchronous — the response returns immediately and the audio is delivered via Telegram when ready.

### TTS API

OpenAI TTS (`tts-1` or `tts-1-hd`) with `opus` response format (OGG/Opus). Max 4096 chars per request — the backend chunks long articles at paragraph/sentence/word boundaries and concatenates the resulting audio files using `ffmpeg`.

### Telegram Bot API Integration

1. Create a bot via @BotFather on Telegram. Save the bot token.
2. The user starts a conversation with the bot to obtain their `chat_id`.
3. The backend uses `sendAudio` (for MP3 with metadata) or `sendVoice` (for OGG, plays inline) to deliver the file.

The backend uses `sendVoice` (OGG/OPUS, plays inline as a voice message). Max 50MB per file.

### Text Preprocessing

Implemented in `backend/processor.go`:

- Strip HTML tags via regex.
- Decode common HTML entities (`&amp;`, `&lt;`, `&gt;`, `&quot;`, `&#39;`, `&nbsp;`).
- Remove URLs (not useful for TTS).
- Normalize whitespace (collapse multiple spaces, limit consecutive newlines).
- Chunk at paragraph boundaries (`\n\n`), then sentence boundaries (`[.!?]\s+`), then word boundaries as fallback. Max 4096 chars per chunk.
- Article title is prepended as a spoken header.

### Server Structure

```
backend/
├── main.go             # Entry point, HTTP server with /api/tts endpoint
├── bot.go              # Telegram bot handlers, message sending
├── tts.go              # OpenAI TTS API client, chunking, ffmpeg concatenation
├── extractor.go        # Server-side article extraction via go-readability
├── processor.go        # Text cleaning and chunking
├── config.go           # Environment variables
├── go.mod
├── .env.example
└── Dockerfile
```

### Environment Variables / Configuration

```
TELEGRAM_BOT_TOKEN=...       # From @BotFather (required)
OPENAI_API_KEY=...           # OpenAI API key (required)
ALLOWED_USER_IDS=...         # Comma-separated Telegram user IDs (empty = allow all)
DEFAULT_CHAT_ID=...          # Default chat ID for addon requests
TTS_VOICE=alloy              # Voice selection (default: alloy)
TTS_MODEL=tts-1              # Model: tts-1 or tts-1-hd (default: tts-1)
SERVER_SECRET=...            # Shared secret for addon HTTP requests (optional)
LISTEN_ADDR=:8080            # HTTP listen address (default: :8080)
```

### Security Considerations

- The backend endpoint should be authenticated to prevent abuse. A simple shared secret/API key in the request header from the addon is sufficient.
- Rate limiting on the endpoint to prevent runaway TTS costs.
- HTTPS is mandatory since you're transmitting article content.

---

## Alternative Flow: Telegram Bot Only (No Addon)

For articles that *don't* have anti-bot protections, the bot also accepts URLs directly:

1. User sends a URL to the Telegram bot.
2. Bot fetches the page server-side using `go-shiori/go-readability`.
3. If extraction succeeds, proceed with TTS and send audio back.
4. If extraction fails (anti-bot), inform the user to use the Firefox addon instead.

This gives the user two paths: the addon for protected sites, and a simple "paste link in Telegram" flow for everything else. This is the primary flow (implemented first).

---

## Data Flow Summary

```
┌──────────────────────────────────────────────────────────┐
│                     USER'S BROWSER                        │
│                                                           │
│  1. User browses to article (past all protections)        │
│  2. Clicks addon button                                   │
│  3. Readability.js extracts text from live DOM             │
│  4. Addon POSTs { title, text, url } to backend           │
└─────────────────────┬─────────────────────────────────────┘
                      │ HTTPS POST
                      ▼
┌──────────────────────────────────────────────────────────┐
│                     BACKEND SERVER                         │
│                                                           │
│  5. Receives text, preprocesses (clean, chunk)            │
│  6. Calls TTS API → receives audio chunks                 │
│  7. Concatenates audio into single file                   │
│  8. Sends audio via Telegram Bot API                      │
└─────────────────────┬─────────────────────────────────────┘
                      │ Telegram Bot API
                      ▼
┌──────────────────────────────────────────────────────────┐
│                     TELEGRAM                               │
│                                                           │
│  9. User receives audio message in their Telegram chat    │
└──────────────────────────────────────────────────────────┘
```

---

## Decisions

1. **TTS Provider**: OpenAI (`tts-1` / `tts-1-hd`, configurable via `TTS_MODEL`)
2. **Audio Format**: OGG/OPUS via `sendVoice` (inline playback in Telegram)
3. **Backend Language**: Go — single binary, `go-telegram-bot-api/v5`, `go-shiori/go-readability`
4. **Hosting**: Self-hosted (Docker or bare binary + ffmpeg)
5. **Authentication**: `ALLOWED_USER_IDS` whitelist (static config). `/start` command shows user their ID for configuration.
6. **Chunking Strategy**: Paragraph → sentence → word boundary splitting, max 4096 chars per chunk, concatenation via ffmpeg
7. **Dual Mode**: Telegram bot accepts URLs directly (primary, implemented first). Firefox addon for protected sites (also implemented).
8. **Addon Distribution**: Personal use only (load via `about:debugging` or unsigned `.xpi`)
