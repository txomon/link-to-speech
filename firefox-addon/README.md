# Firefox Addon

Firefox WebExtension that extracts article text from the current page using Mozilla's Readability.js and sends it to the backend for TTS conversion.

## Why an addon?

Many websites use anti-bot protections (Cloudflare challenges, JS rendering walls, cookie consent gates, login-gated paywalls) that prevent server-side scraping. The addon operates on the fully-rendered DOM in a real browser session — the user has already navigated past all protections.

## Installation (personal use)

1. Open `about:debugging#/runtime/this-firefox` in Firefox
2. Click "Load Temporary Add-on..."
3. Select `manifest.json` from this directory

Note: Temporary add-ons are removed when Firefox restarts. For persistent installation, use `about:addons` → gear icon → "Install Add-on From File..." with a signed `.xpi`, or set `xpinstall.signatures.required` to `false` in `about:config` (Nightly/Developer Edition only).

## Configuration

Click the addon's preferences (via `about:addons`) to set:

- **Server URL** — Backend endpoint (default: `http://localhost:8080/api/tts`)
- **Server Secret** — Shared secret matching the backend's `SERVER_SECRET`
- **Telegram Chat ID** — Your chat ID (optional if backend has `DEFAULT_CHAT_ID`)

## Usage

1. Navigate to any article in Firefox
2. Click the toolbar icon
3. The addon extracts the article text and sends it to the backend
4. You receive a voice message in Telegram

## Icons

Place 48x48 and 96x96 PNG icons in `icons/icon-48.png` and `icons/icon-96.png`. The addon works without them (uses a default icon).
