const DEFAULT_SERVER_URL = "http://localhost:8080/api/tts";

async function getSettings() {
  return browser.storage.local.get({
    serverUrl: DEFAULT_SERVER_URL,
    serverSecret: "",
    chatId: "",
  });
}

browser.browserAction.onClicked.addListener(async (tab) => {
  try {
    await browser.tabs.executeScript(tab.id, { file: "readability.js" });
    await browser.tabs.executeScript(tab.id, { file: "content-script.js" });
  } catch (err) {
    console.error("Injection failed:", err);
    browser.notifications.create({
      type: "basic",
      title: "Article to Audio",
      message: "Failed to extract article. Make sure you're on a regular web page.",
    });
  }
});

browser.runtime.onMessage.addListener(async (msg) => {
  if (msg.type === "article_extracted") {
    const settings = await getSettings();

    const headers = { "Content-Type": "application/json" };
    if (settings.serverSecret) {
      headers["Authorization"] = "Bearer " + settings.serverSecret;
    }

    const body = {
      title: msg.title,
      text: msg.text,
      url: msg.url,
    };
    if (settings.chatId) {
      body.chat_id = parseInt(settings.chatId, 10);
    }

    try {
      const resp = await fetch(settings.serverUrl, {
        method: "POST",
        headers,
        body: JSON.stringify(body),
      });

      if (resp.ok) {
        browser.notifications.create({
          type: "basic",
          title: "Article to Audio",
          message: `Sent "${msg.title}" for audio conversion.`,
        });
      } else {
        const errText = await resp.text();
        browser.notifications.create({
          type: "basic",
          title: "Article to Audio - Error",
          message: `Server error ${resp.status}: ${errText}`,
        });
      }
    } catch (err) {
      browser.notifications.create({
        type: "basic",
        title: "Article to Audio - Error",
        message: `Cannot reach server: ${err.message}`,
      });
    }
  } else if (msg.type === "extraction_failed") {
    browser.notifications.create({
      type: "basic",
      title: "Article to Audio",
      message: "Could not extract article content from this page.",
    });
  }
});
