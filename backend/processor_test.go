package main

import (
	"strings"
	"testing"
)

func TestCleanText_StripHTML(t *testing.T) {
	input := "<p>Hello <b>world</b></p>"
	got := CleanText(input)
	if strings.Contains(got, "<") || strings.Contains(got, ">") {
		t.Errorf("CleanText did not strip HTML tags: %q", got)
	}
	if got != "Hello world" {
		t.Errorf("CleanText(%q) = %q, want %q", input, got, "Hello world")
	}
}

func TestCleanText_DecodeEntities(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"Tom &amp; Jerry", "Tom & Jerry"},
		{"&lt;script&gt;", "<script>"},
		{"She said &quot;hello&quot;", `She said "hello"`},
		{"it&#39;s fine", "it's fine"},
		{"non&nbsp;breaking", "non breaking"},
	}
	for _, tt := range tests {
		got := CleanText(tt.input)
		if got != tt.want {
			t.Errorf("CleanText(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestCleanText_RemoveURLs(t *testing.T) {
	input := "Visit http://example.com or https://example.org/path?q=1 for info."
	got := CleanText(input)
	if strings.Contains(got, "http") {
		t.Errorf("CleanText did not remove URLs: %q", got)
	}
}

func TestCleanText_NormalizeWhitespace(t *testing.T) {
	input := "hello    world\t\ttab"
	got := CleanText(input)
	if got != "hello world tab" {
		t.Errorf("CleanText(%q) = %q, want %q", input, got, "hello world tab")
	}
}

func TestCleanText_CollapseNewlines(t *testing.T) {
	input := "para one\n\n\n\n\npara two"
	got := CleanText(input)
	if got != "para one\n\npara two" {
		t.Errorf("CleanText(%q) = %q, want %q", input, got, "para one\n\npara two")
	}
}

func TestCleanText_Empty(t *testing.T) {
	got := CleanText("")
	if got != "" {
		t.Errorf("CleanText(\"\") = %q, want %q", got, "")
	}
}

func TestCleanText_TrimWhitespace(t *testing.T) {
	input := "  \n  hello  \n  "
	got := CleanText(input)
	if got != "hello" {
		t.Errorf("CleanText(%q) = %q, want %q", input, got, "hello")
	}
}

func TestChunkText_ShortText(t *testing.T) {
	text := "Hello world."
	chunks := ChunkText(text, 100)
	if len(chunks) != 1 {
		t.Fatalf("expected 1 chunk, got %d", len(chunks))
	}
	if chunks[0] != text {
		t.Errorf("chunk = %q, want %q", chunks[0], text)
	}
}

func TestChunkText_SplitByParagraph(t *testing.T) {
	para1 := strings.Repeat("a", 50)
	para2 := strings.Repeat("b", 50)
	text := para1 + "\n\n" + para2
	chunks := ChunkText(text, 60)
	if len(chunks) != 2 {
		t.Fatalf("expected 2 chunks, got %d: %v", len(chunks), chunks)
	}
	if chunks[0] != para1 {
		t.Errorf("chunk[0] = %q, want %q", chunks[0], para1)
	}
	if chunks[1] != para2 {
		t.Errorf("chunk[1] = %q, want %q", chunks[1], para2)
	}
}

func TestChunkText_SplitBySentence(t *testing.T) {
	// One long paragraph that exceeds maxLen, force sentence-level split
	s1 := strings.Repeat("x", 40) + "."
	s2 := strings.Repeat("y", 40) + "."
	text := s1 + " " + s2 // single paragraph, 83 chars total
	chunks := ChunkText(text, 50)
	if len(chunks) != 2 {
		t.Fatalf("expected 2 chunks, got %d: %v", len(chunks), chunks)
	}
}

func TestChunkText_SplitByWord(t *testing.T) {
	// One long sentence (no period/split points), force word-level split
	words := make([]string, 20)
	for i := range words {
		words[i] = "word"
	}
	text := strings.Join(words, " ") // "word word word..." = 99 chars
	chunks := ChunkText(text, 30)
	for _, c := range chunks {
		if len(c) > 30 {
			t.Errorf("chunk exceeds maxLen: len=%d, chunk=%q", len(c), c)
		}
	}
	// Reassembled text should match original
	reassembled := strings.Join(chunks, " ")
	if reassembled != text {
		t.Errorf("reassembled text doesn't match original")
	}
}

func TestChunkText_EmptyText(t *testing.T) {
	chunks := ChunkText("", 100)
	if len(chunks) != 1 || chunks[0] != "" {
		t.Errorf("ChunkText(\"\", 100) = %v, want [\"\"]", chunks)
	}
}

func TestChunkText_MaxChunkSizeConstant(t *testing.T) {
	// Verify the constant is set correctly for OpenAI TTS
	if maxChunkSize != 4096 {
		t.Errorf("maxChunkSize = %d, want 4096", maxChunkSize)
	}
}

func TestChunkText_UnicodeText(t *testing.T) {
	// Unicode characters (multi-byte) should be handled
	text := strings.Repeat("日本語テスト。", 20)
	chunks := ChunkText(text, 50)
	for _, c := range chunks {
		if c == "" {
			t.Error("got empty chunk")
		}
	}
}

func TestChunkText_AllChunksNonEmpty(t *testing.T) {
	text := "First paragraph.\n\n\n\nSecond paragraph.\n\n\n\nThird paragraph."
	chunks := ChunkText(text, 30)
	for i, c := range chunks {
		if strings.TrimSpace(c) == "" {
			t.Errorf("chunk[%d] is empty or whitespace-only", i)
		}
	}
}
