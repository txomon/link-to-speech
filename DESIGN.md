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
├── readability.js          # Mozilla Readability.js library (vendored)
├── icons/
│   ├── icon-48.png
│   └── icon-96.png
└── options.html (optional) # Settings page for server URL, Telegram chat ID, etc.
```

### manifest.json Permissions Required

```json
{
  "manifest_version": 2,
  "name": "Article to TTS",
  "version": "1.0",
  "permissions": [
    "activeTab",
    "scripting"
  ],
  "browser_action": {
    "default_icon": "icons/icon-48.png",
    "default_title": "Send article to TTS"
  },
  "background": {
    "scripts": ["background.js"]
  }
}
```

- `activeTab` is sufficient — no need for broad host permissions. It grants access to the current tab only when the user clicks the button.
- No need for `<all_urls>` or specific host patterns.

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

### Tech Stack Options

- **Python** (recommended): Flask or FastAPI for the HTTP endpoint, `python-telegram-bot` or `aiogram` for Telegram integration.
- **Node.js** (alternative): Express for the HTTP endpoint, `node-telegram-bot-api` or `telegraf` for Telegram.

### Flow

1. Receive POST request from the Firefox addon with `{ title, text, url }`.
2. (Optional) Preprocess text: remove excessive whitespace, truncate if too long for TTS API limits, split into chunks if needed.
3. Call TTS API with the text.
4. Receive audio file (typically MP3 or OGG).
5. Send audio to the user's Telegram chat via the Telegram Bot API.

### API Endpoint

```
POST /api/tts
Content-Type: application/json

{
  "title": "Article Title",
  "text": "Full article plain text...",
  "url": "https://example.com/article"
}

Response: 200 OK { "status": "sent" }
```

### TTS API

The specific TTS provider is a user choice. Common options:

| Provider | Notes |
|----------|-------|
| OpenAI TTS (`tts-1`, `tts-1-hd`) | High quality, simple API, supports multiple voices. Max ~4096 chars per request — chunking needed for long articles. |
| ElevenLabs | Very natural voices, streaming support, generous free tier. |
| Google Cloud TTS | Wide language support, WaveNet voices. |
| Azure Cognitive Services TTS | Neural voices, SSML support. |
| Coqui TTS / Piper (self-hosted) | Free, runs locally, no API costs. Lower quality. |

**Important:** Most TTS APIs have per-request character limits. The backend must handle chunking long articles into segments, generating audio for each, and concatenating the resulting audio files (e.g., using `pydub` or `ffmpeg`).

### Telegram Bot API Integration

1. Create a bot via @BotFather on Telegram. Save the bot token.
2. The user starts a conversation with the bot to obtain their `chat_id`.
3. The backend uses `sendAudio` (for MP3 with metadata) or `sendVoice` (for OGG, plays inline) to deliver the file.

```python
# Example using python-telegram-bot
import telegram

bot = telegram.Bot(token=BOT_TOKEN)
bot.send_audio(
    chat_id=USER_CHAT_ID,
    audio=open("article.mp3", "rb"),
    title=article_title,
    caption=f"🔊 {article_title}\n{article_url}"
)
```

- `sendAudio`: Supports title, performer, duration metadata. File appears as a music track. Max 50MB.
- `sendVoice`: Plays inline like a voice message. Must be OGG with OPUS codec. Max 50MB.

### Text Preprocessing

Before sending to TTS:

- Strip any remaining HTML tags if present.
- Remove or replace characters that cause TTS issues (e.g., excessive URLs, code blocks, special unicode).
- Normalize whitespace.
- If text exceeds TTS API limits, split at sentence boundaries (use nltk, spacy, or regex-based sentence splitting).
- Optionally prepend the article title as a spoken header.

### Server Structure (Python Example)

```
backend/
├── main.py                 # FastAPI/Flask app with /api/tts endpoint
├── tts_service.py          # TTS API integration, chunking, audio concatenation
├── telegram_service.py     # Telegram bot message sending
├── text_processor.py       # Text cleaning and preprocessing
├── config.py               # Environment variables, API keys
├── requirements.txt
└── Dockerfile (optional)
```

### Environment Variables / Configuration

```
TTS_API_KEY=...              # TTS provider API key
TTS_PROVIDER=openai          # Which TTS service to use
TTS_VOICE=alloy              # Voice selection
TELEGRAM_BOT_TOKEN=...       # From @BotFather
TELEGRAM_CHAT_ID=...         # Target chat to send audio to
SERVER_SECRET=...            # (Optional) Shared secret to authenticate addon requests
```

### Security Considerations

- The backend endpoint should be authenticated to prevent abuse. A simple shared secret/API key in the request header from the addon is sufficient.
- Rate limiting on the endpoint to prevent runaway TTS costs.
- HTTPS is mandatory since you're transmitting article content.

---

## Alternative Flow: Telegram Bot Only (No Addon)

For articles that *don't* have anti-bot protections, the bot can also accept URLs directly:

1. User sends a URL to the Telegram bot.
2. Bot fetches the page server-side using `trafilatura` (Python) or `@mozilla/readability` + `jsdom` (Node.js).
3. If extraction succeeds, proceed with TTS and send audio back.
4. If extraction fails (anti-bot), inform the user to use the Firefox addon instead.

This gives the user two paths: the addon for protected sites, and a simple "paste link in Telegram" flow for everything else.

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

## other decisions

1. **TTS Provider**: Openai
2. **Audio Format**: OGG/OPUS via `sendVoice` (inline playback)?
3. **Backend Language**: golang
4. **Hosting**: self hosted
5. **Authentication**: Log in with telegram into the bot
6. **Chunking Strategy**: How to split by paragraph or section if necessary
7. **Dual Mode**: Should the Telegram bot also accept direct URL messages for server-side extraction as a fallback, and that should be the first part to implement, addon would come afterwards
8. **Addon Distribution**: Personal use only
