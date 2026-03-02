(function () {
  const clone = document.cloneNode(true);
  const article = new Readability(clone).parse();

  if (article && article.textContent) {
    browser.runtime.sendMessage({
      type: "article_extracted",
      title: article.title || document.title,
      text: article.textContent,
      excerpt: article.excerpt,
      byline: article.byline,
      siteName: article.siteName,
      url: window.location.href,
    });
  } else {
    browser.runtime.sendMessage({
      type: "extraction_failed",
      url: window.location.href,
    });
  }
})();
