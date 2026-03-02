package main

import (
	"regexp"
	"strings"
	"unicode"
)

const maxChunkSize = 4096

var (
	htmlTagRe    = regexp.MustCompile(`<[^>]*>`)
	urlRe        = regexp.MustCompile(`https?://\S+`)
	multiSpaceRe = regexp.MustCompile(`[^\S\n]+`)
	multiNewlRe  = regexp.MustCompile(`\n{3,}`)
	sentenceRe   = regexp.MustCompile(`([.!?])\s+`)
)

// CleanText strips HTML tags, removes URLs, and normalizes whitespace.
func CleanText(text string) string {
	text = htmlTagRe.ReplaceAllString(text, "")

	// Decode common HTML entities
	r := strings.NewReplacer(
		"&amp;", "&",
		"&lt;", "<",
		"&gt;", ">",
		"&quot;", `"`,
		"&#39;", "'",
		"&nbsp;", " ",
	)
	text = r.Replace(text)

	text = urlRe.ReplaceAllString(text, "")
	text = multiSpaceRe.ReplaceAllString(text, " ")
	text = multiNewlRe.ReplaceAllString(text, "\n\n")

	lines := strings.Split(text, "\n")
	for i, line := range lines {
		lines[i] = strings.TrimSpace(line)
	}
	text = strings.Join(lines, "\n")

	return strings.TrimSpace(text)
}

// ChunkText splits text into chunks not exceeding maxLen characters,
// splitting at paragraph boundaries first, then sentence, then word.
func ChunkText(text string, maxLen int) []string {
	if len(text) <= maxLen {
		return []string{text}
	}

	paragraphs := strings.Split(text, "\n\n")
	var chunks []string
	var cur strings.Builder

	for _, para := range paragraphs {
		para = strings.TrimSpace(para)
		if para == "" {
			continue
		}

		if cur.Len() > 0 && cur.Len()+len(para)+2 > maxLen {
			chunks = append(chunks, strings.TrimSpace(cur.String()))
			cur.Reset()
		}

		// Single paragraph exceeds limit — split deeper
		if len(para) > maxLen {
			if cur.Len() > 0 {
				chunks = append(chunks, strings.TrimSpace(cur.String()))
				cur.Reset()
			}
			chunks = append(chunks, chunkBySentences(para, maxLen)...)
			continue
		}

		if cur.Len() > 0 {
			cur.WriteString("\n\n")
		}
		cur.WriteString(para)
	}

	if cur.Len() > 0 {
		chunks = append(chunks, strings.TrimSpace(cur.String()))
	}

	return chunks
}

func chunkBySentences(text string, maxLen int) []string {
	// Split text but keep the delimiters (sentence-ending punctuation)
	parts := splitKeepDelimiter(text, sentenceRe)

	var chunks []string
	var cur strings.Builder

	for _, sent := range parts {
		sent = strings.TrimSpace(sent)
		if sent == "" {
			continue
		}

		if cur.Len() > 0 && cur.Len()+len(sent)+1 > maxLen {
			chunks = append(chunks, strings.TrimSpace(cur.String()))
			cur.Reset()
		}

		if len(sent) > maxLen {
			if cur.Len() > 0 {
				chunks = append(chunks, strings.TrimSpace(cur.String()))
				cur.Reset()
			}
			chunks = append(chunks, chunkByWords(sent, maxLen)...)
			continue
		}

		if cur.Len() > 0 {
			cur.WriteString(" ")
		}
		cur.WriteString(sent)
	}

	if cur.Len() > 0 {
		chunks = append(chunks, strings.TrimSpace(cur.String()))
	}

	return chunks
}

// splitKeepDelimiter splits text by a regex but reattaches the delimiter
// to the preceding segment.
func splitKeepDelimiter(text string, re *regexp.Regexp) []string {
	indices := re.FindAllStringIndex(text, -1)
	if len(indices) == 0 {
		return []string{text}
	}

	var parts []string
	prev := 0
	for _, loc := range indices {
		// Include the punctuation character (first byte of match) with the segment
		end := loc[0] + 1
		parts = append(parts, text[prev:end])
		prev = loc[1]
	}
	if prev < len(text) {
		parts = append(parts, text[prev:])
	}
	return parts
}

func chunkByWords(text string, maxLen int) []string {
	words := strings.FieldsFunc(text, unicode.IsSpace)
	var chunks []string
	var cur strings.Builder

	for _, w := range words {
		if cur.Len() > 0 && cur.Len()+len(w)+1 > maxLen {
			chunks = append(chunks, cur.String())
			cur.Reset()
		}
		if cur.Len() > 0 {
			cur.WriteString(" ")
		}
		cur.WriteString(w)
	}

	if cur.Len() > 0 {
		chunks = append(chunks, cur.String())
	}

	return chunks
}
