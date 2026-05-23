package llm

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/sanja/octocli_cg/internal/config"
)

const defaultHTTPTimeout = 5 * time.Minute

type OpenAICompatibleClient struct {
	profile    config.Profile
	httpClient *http.Client
}

type openAIChatRequest struct {
	Model       string    `json:"model"`
	Messages    []Message `json:"messages"`
	Stream      bool      `json:"stream"`
	Temperature *float64  `json:"temperature,omitempty"`
	MaxTokens   *int      `json:"max_tokens,omitempty"`
}

type streamChunk struct {
	Choices []struct {
		Delta struct {
			Content string `json:"content"`
		} `json:"delta"`
	} `json:"choices"`
	Error *struct {
		Message string `json:"message"`
		Type    string `json:"type"`
	} `json:"error,omitempty"`
}

func NewOpenAICompatibleClient(profile config.Profile) *OpenAICompatibleClient {
	return &OpenAICompatibleClient{
		profile: profile,
		httpClient: &http.Client{
			Timeout: defaultHTTPTimeout,
		},
	}
}

func (c *OpenAICompatibleClient) StreamChat(ctx context.Context, request ChatRequest, onDelta StreamFunc) error {
	if len(request.Messages) == 0 {
		return errors.New("chat request requires at least one message")
	}
	if onDelta == nil {
		return errors.New("stream callback is required")
	}

	payload := openAIChatRequest{
		Model:       c.profile.Model,
		Messages:    request.Messages,
		Stream:      true,
		Temperature: request.Temperature,
		MaxTokens:   request.MaxTokens,
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal chat request: %w", err)
	}

	endpoint := strings.TrimRight(c.profile.BaseURL, "/") + "/chat/completions"
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("create chat request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Accept", "text/event-stream")
	apiKey := expandEnv(c.profile.APIKey)
	if strings.TrimSpace(apiKey) != "" {
		httpReq.Header.Set("Authorization", "Bearer "+apiKey)
	}

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return fmt.Errorf("send chat request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		responseBody, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return fmt.Errorf("llm endpoint returned %s: %s", resp.Status, strings.TrimSpace(string(responseBody)))
	}

	return readServerSentEvents(resp.Body, onDelta)
}

func readServerSentEvents(reader io.Reader, onDelta StreamFunc) error {
	scanner := bufio.NewScanner(reader)
	buffer := make([]byte, 0, 64*1024)
	scanner.Buffer(buffer, 1024*1024)

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, ":") {
			continue
		}
		if !strings.HasPrefix(line, "data:") {
			continue
		}

		data := strings.TrimSpace(strings.TrimPrefix(line, "data:"))
		if data == "[DONE]" {
			return nil
		}

		var chunk streamChunk
		if err := json.Unmarshal([]byte(data), &chunk); err != nil {
			return fmt.Errorf("parse stream chunk: %w", err)
		}
		if chunk.Error != nil {
			return fmt.Errorf("llm stream error: %s (%s)", chunk.Error.Message, chunk.Error.Type)
		}
		for _, choice := range chunk.Choices {
			if choice.Delta.Content == "" {
				continue
			}
			if err := onDelta(choice.Delta.Content); err != nil {
				return err
			}
		}
	}
	if err := scanner.Err(); err != nil {
		return fmt.Errorf("read stream: %w", err)
	}
	return nil
}

func expandEnv(value string) string {
	return os.ExpandEnv(value)
}
