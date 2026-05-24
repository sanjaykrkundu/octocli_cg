package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/sanja/octocli_cg/internal/llm"
	"github.com/sanja/octocli_cg/internal/tools"
)

const DefaultMaxIterations = 8

type Loop struct {
	Client        llm.Client
	Tools         tools.Runtime
	MaxIterations int
	SystemPrompt  string
	Rules         string
	Depth         int
	MaxDepth      int
}

type ModelResponse struct {
	Thought     string      `json:"thought,omitempty"`
	ToolCall    *tools.Call `json:"tool_call,omitempty"`
	FinalAnswer string      `json:"final_answer,omitempty"`
}

func (l Loop) Run(ctx context.Context, userPrompt string) (string, error) {
	maxIterations := l.MaxIterations
	if maxIterations <= 0 {
		maxIterations = DefaultMaxIterations
	}
	messages := []llm.Message{
		{Role: "system", Content: l.systemPrompt()},
		{Role: "user", Content: userPrompt},
	}

	for i := 0; i < maxIterations; i++ {
		raw, err := l.complete(ctx, messages)
		if err != nil {
			return "", err
		}

		var response ModelResponse
		if err := json.Unmarshal([]byte(raw), &response); err != nil {
			return "", fmt.Errorf("parse model response as JSON: %w\nraw response: %s", err, raw)
		}

		messages = append(messages, llm.Message{Role: "assistant", Content: raw})

		if strings.TrimSpace(response.FinalAnswer) != "" {
			return response.FinalAnswer, nil
		}
		if response.ToolCall == nil {
			return "", fmt.Errorf("model response did not include tool_call or final_answer")
		}

		result := l.Tools.Execute(ctx, *response.ToolCall)
		resultJSON, err := json.Marshal(result)
		if err != nil {
			return "", fmt.Errorf("marshal tool result: %w", err)
		}

		messages = append(messages, llm.Message{
			Role:    "user",
			Content: "TOOL_RESULT:\n" + string(resultJSON),
		})
	}

	return "", fmt.Errorf("agent loop reached max iterations (%d) without final_answer", maxIterations)
}

func (l Loop) complete(ctx context.Context, messages []llm.Message) (string, error) {
	var builder strings.Builder
	err := l.Client.StreamChat(ctx, llm.ChatRequest{Messages: messages}, func(delta string) error {
		builder.WriteString(delta)
		return nil
	})
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(builder.String()), nil
}

func (l Loop) systemPrompt() string {
	if strings.TrimSpace(l.SystemPrompt) != "" {
		return l.SystemPrompt
	}

	definitions, _ := json.MarshalIndent(l.Tools.Definitions(), "", "  ")
	prompt := fmt.Sprintf(`You are octocli_cg, a CLI coding agent using a ReAct-style loop.

You must respond with valid JSON only.
Return exactly one of these shapes:

1. To use a tool:
{
  "thought": "brief reasoning",
  "tool_call": {
    "name": "read_file | write_file | execute_shell",
    "arguments": { ... }
  }
}

2. To finish:
{
  "thought": "brief reasoning",
  "final_answer": "your final answer"
}

Rules:
- Never output markdown fences.
- Use tools when you need workspace or command information.
- After receiving TOOL_RESULT, continue reasoning from that result.
- Keep thoughts brief.
- Current delegation depth: %d.

Available tools:
%s`, l.Depth, string(definitions))
	if l.MaxDepth > 0 {
		prompt += fmt.Sprintf("\nMaximum delegation depth: %d.", l.MaxDepth)
	}
	if strings.TrimSpace(l.Rules) == "" {
		return prompt
	}
	return prompt + "\n\nAdditional rules context:\n" + l.Rules
}
