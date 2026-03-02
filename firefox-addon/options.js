const serverUrlInput = document.getElementById("serverUrl");
const serverSecretInput = document.getElementById("serverSecret");
const chatIdInput = document.getElementById("chatId");
const saveBtn = document.getElementById("save");
const statusEl = document.getElementById("status");

browser.storage.local
  .get({ serverUrl: "http://localhost:8080/api/tts", serverSecret: "", chatId: "" })
  .then((s) => {
    serverUrlInput.value = s.serverUrl;
    serverSecretInput.value = s.serverSecret;
    chatIdInput.value = s.chatId;
  });

saveBtn.addEventListener("click", () => {
  browser.storage.local
    .set({
      serverUrl: serverUrlInput.value,
      serverSecret: serverSecretInput.value,
      chatId: chatIdInput.value,
    })
    .then(() => {
      statusEl.hidden = false;
      setTimeout(() => { statusEl.hidden = true; }, 2000);
    });
});
