# Project Status & TODO

## What has been done

### Go Backend (`backend/`)

- **config.go** — Loads all configuration from environment variables: `TELEGRAM_BOT_TOKEN`, `OPENAI_API_KEY`, `ALLOWED_USER_IDS`, `DEFAULT_CHAT_ID`, `TTS_VOICE`, `TTS_MODEL`, `SERVER_SECRET`, `LISTEN_ADDR`. Sensible defaults for optional values.
- **processor.go** — Text cleaning (strips HTML tags, removes URLs, normalizes whitespace, decodes HTML entities) and chunking (splits at paragraph → sentence → word boundaries, respecting the 4096-char OpenAI TTS limit). Pre-compiled regexes.
- **extractor.go** — Server-side article extraction using `go-shiori/go-readability`. Fetches URL with a browser-like User-Agent, parses into clean plain text. This is the "dual mode" path for when the user sends a URL directly to the Telegram bot.
- **tts.go** — OpenAI TTS client. Calls `POST /v1/audio/speech` with `opus` response format. For long articles, generates audio per chunk and concatenates with ffmpeg (`-c copy` with re-encoding fallback). Temp files cleaned up automatically.
- **bot.go** — Telegram bot with: `/start` (shows chat/user ID), `/help`, URL auto-detection in messages, user authorization via `ALLOWED_USER_IDS`, progress status messages (extract → generate → send → delete status), `sendVoice` for inline OGG/Opus playback. Also exposes `ProcessArticle()` for the HTTP API path.
- **main.go** — Entry point. Starts both the Telegram bot (long polling) and an HTTP server. `POST /api/tts` endpoint for the Firefox addon with `Bearer` token auth, JSON body `{title, text, url, chat_id}`. ffmpeg availability check at startup.
- **Dockerfile** — Multi-stage: `golang:1.22-alpine` build → `alpine:3.19` runtime with `ffmpeg` and `ca-certificates`.
- **.env.example** — Template with all config variables documented.
- **README.md** — Setup and usage docs.
- **Build verified** — `go build` and `go vet` pass clean.

### Firefox Addon (`firefox-addon/`)

- **manifest.json** — MV2, permissions: `activeTab`, `storage`, `notifications`. Browser action with toolbar icon.
- **background.js** — On toolbar click, injects `readability.js` then `content-script.js` into the active tab. Listens for extracted article messages, POSTs to backend with configurable server URL/secret/chat ID from `browser.storage.local`. Shows browser notifications on success/failure.
- **content-script.js** — Clones the DOM, runs `new Readability(clone).parse()`, sends `{title, textContent, excerpt, byline, siteName, url}` back via `browser.runtime.sendMessage()`.
- **readability.js** — Vendored from `@mozilla/readability` npm package (2786 lines, Apache 2.0 license).
- **options.html + options.js** — Settings page for server URL, server secret, and Telegram chat ID.
- **README.md** — Installation and usage docs.

### CI/CD (`.github/workflows/`)

- **ci.yml** — Runs on push to master/main and PRs. Jobs: Go build + vet, addon manifest validation + xpi packaging.
- **release.yml** — Runs on `v*` tags. Cross-compiles Go backend for linux/darwin × amd64/arm64 (4 binaries, no compression per spec). Packages addon as `.xpi` zip. Creates GitHub release with all artifacts via `softprops/action-gh-release`.

### Repo Structure

- Monorepo with `backend/` and `firefox-addon/` at the root.
- `.gitignore` — Ignores binary, `.env`, IDE files, OS files.
- `README.md` — Root README with architecture diagram and quick start.
- `DESIGN.md` — Original design document (added by user).
- 7 commits with sensible granularity, rebased on top of the user's `design.md` commit.

---

## What is NOT done yet / TODO

### Testing

- [ ] No unit tests for any Go code. Priority targets:
  - `processor.go` — `CleanText()` and `ChunkText()` with edge cases (empty text, single huge paragraph, unicode, HTML entities)
  - `tts.go` — mock OpenAI API responses, verify chunking/concatenation logic
  - `bot.go` — mock Telegram API, verify URL extraction regex
- [ ] No integration tests (end-to-end with real APIs)
- [ ] No test for the Firefox addon

### Backend

- [ ] **No graceful shutdown** — the HTTP server doesn't get a proper `Shutdown()` call with context; it just exits
- [ ] **No rate limiting** on the `/api/tts` endpoint (mentioned in design as needed to prevent runaway TTS costs)
- [ ] **No request size limit** — should cap the incoming JSON body size to prevent abuse
- [ ] **Telegram `sendVoice` 50MB limit** — no handling for very long articles that could exceed this; should split into multiple voice messages or warn the user
- [ ] **No retry logic** for OpenAI API calls (transient failures, rate limits)
- [ ] **go-readability is deprecated** — the library prints a deprecation warning suggesting `codeberg.org/readeck/go-readability/v2`. Should migrate.
- [ ] **No health check endpoint** (useful for Docker/k8s deployments)
- [ ] **CORS headers** not set on `/api/tts` — not needed for the addon (background script fetch), but would be needed if ever called from a web page

### Firefox Addon

- [ ] **No icons** — `icons/icon-48.png` and `icons/icon-96.png` are referenced in manifest but don't exist. Addon works with default icon but looks unfinished.
- [ ] **No visual feedback during processing** — badge text or icon change while the request is in flight would improve UX
- [ ] **No error handling for Readability failure modes** — some pages may partially parse; should handle gracefully
- [ ] **MV2 only** — Firefox still supports MV2 but MV3 migration may be needed eventually
- [ ] **Not packaged as signed .xpi** — for persistent installation beyond `about:debugging` temporary loading, needs either signing via AMO or `xpinstall.signatures.required = false`

### CI/CD

- [ ] **Release workflow not tested** — no tags have been pushed yet to verify it works end-to-end
- [ ] **No Docker image build/push** in CI — could add a job to build and push to GHCR
- [ ] **No linting** — could add `golangci-lint` for Go code
- [ ] **Addon .xpi not signed** in the release workflow

### Deployment

- [ ] **No docker-compose.yml** — would be convenient for self-hosting with env file
- [ ] **No systemd service file** — alternative to Docker for bare-metal deployment
- [ ] **HTTPS not configured** — the HTTP server listens plain; needs a reverse proxy (nginx/caddy) or built-in TLS for production use with the addon

### Design Doc Items Not Yet Addressed

- [ ] **"Authentication: Log in with telegram into the bot"** — current implementation uses a static `ALLOWED_USER_IDS` list. The design mentions Telegram-based login. Could implement a `/register` command with a one-time code, or just use the current approach (sufficient for personal use).
- [ ] **Chunking strategy** — current implementation splits by paragraph then sentence then word. The design mentions splitting by "paragraph or section if necessary". Current approach is functional but could be improved with smarter section-aware splitting.
