package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

type TTSClient struct {
	apiKey string
	model  string
	voice  string
	client *http.Client
}

func NewTTSClient(apiKey, model, voice string) *TTSClient {
	return &TTSClient{
		apiKey: apiKey,
		model:  model,
		voice:  voice,
		client: &http.Client{Timeout: 120 * time.Second},
	}
}

// GenerateAudio converts text to OGG/Opus audio, handling chunking for long texts.
func (t *TTSClient) GenerateAudio(ctx context.Context, text string) ([]byte, error) {
	chunks := ChunkText(text, maxChunkSize)

	if len(chunks) == 1 {
		return t.generateChunk(ctx, chunks[0])
	}

	tmpDir, err := os.MkdirTemp("", "tts-*")
	if err != nil {
		return nil, fmt.Errorf("creating temp dir: %w", err)
	}
	defer os.RemoveAll(tmpDir)

	var chunkFiles []string
	for i, chunk := range chunks {
		log.Printf("Generating TTS chunk %d/%d (%d chars)", i+1, len(chunks), len(chunk))

		audio, err := t.generateChunk(ctx, chunk)
		if err != nil {
			return nil, fmt.Errorf("chunk %d/%d: %w", i+1, len(chunks), err)
		}

		p := filepath.Join(tmpDir, fmt.Sprintf("chunk_%03d.opus", i))
		if err := os.WriteFile(p, audio, 0644); err != nil {
			return nil, fmt.Errorf("writing chunk %d: %w", i+1, err)
		}
		chunkFiles = append(chunkFiles, p)
	}

	return t.concatenateAudio(ctx, tmpDir, chunkFiles)
}

func (t *TTSClient) generateChunk(ctx context.Context, text string) ([]byte, error) {
	payload, err := json.Marshal(map[string]string{
		"model":           t.model,
		"input":           text,
		"voice":           t.voice,
		"response_format": "opus",
	})
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, "POST",
		"https://api.openai.com/v1/audio/speech", bytes.NewReader(payload))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+t.apiKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := t.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("TTS request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("TTS API %d: %s", resp.StatusCode, string(body))
	}

	return io.ReadAll(resp.Body)
}

func (t *TTSClient) concatenateAudio(ctx context.Context, tmpDir string, files []string) ([]byte, error) {
	// Build ffmpeg concat list
	listPath := filepath.Join(tmpDir, "list.txt")
	var list strings.Builder
	for _, f := range files {
		fmt.Fprintf(&list, "file '%s'\n", f)
	}
	if err := os.WriteFile(listPath, []byte(list.String()), 0644); err != nil {
		return nil, err
	}

	outputPath := filepath.Join(tmpDir, "output.ogg")

	// Try lossless concat (stream copy)
	cmd := exec.CommandContext(ctx, "ffmpeg", "-y",
		"-f", "concat", "-safe", "0",
		"-i", listPath,
		"-c", "copy",
		outputPath,
	)
	if out, err := cmd.CombinedOutput(); err != nil {
		log.Printf("Lossless concat failed, re-encoding: %s", string(out))

		// Fallback: re-encode with libopus
		cmd2 := exec.CommandContext(ctx, "ffmpeg", "-y",
			"-f", "concat", "-safe", "0",
			"-i", listPath,
			"-c:a", "libopus", "-b:a", "64k",
			outputPath,
		)
		if out2, err2 := cmd2.CombinedOutput(); err2 != nil {
			return nil, fmt.Errorf("ffmpeg failed: %s: %w", string(out2), err2)
		}
	}

	return os.ReadFile(outputPath)
}
