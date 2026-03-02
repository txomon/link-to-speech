package main

import (
	"context"
	"fmt"
	"net/http"
	nurl "net/url"
	"time"

	readability "github.com/go-shiori/go-readability"
)

type Article struct {
	Title   string
	Content string
	Excerpt string
	Byline  string
}

func ExtractArticle(ctx context.Context, rawURL string) (*Article, error) {
	parsedURL, err := nurl.Parse(rawURL)
	if err != nil {
		return nil, fmt.Errorf("invalid URL: %w", err)
	}

	client := &http.Client{Timeout: 30 * time.Second}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (X11; Linux x86_64; rv:128.0) Gecko/20100101 Firefox/128.0")

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetching URL: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP %d for %s", resp.StatusCode, rawURL)
	}

	article, err := readability.FromReader(resp.Body, parsedURL)
	if err != nil {
		return nil, fmt.Errorf("parsing article: %w", err)
	}

	if article.TextContent == "" {
		return nil, fmt.Errorf("no article content found at %s", rawURL)
	}

	return &Article{
		Title:   article.Title,
		Content: article.TextContent,
		Excerpt: article.Excerpt,
		Byline:  article.Byline,
	}, nil
}
