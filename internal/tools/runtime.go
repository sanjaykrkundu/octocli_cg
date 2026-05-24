package tools

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

type Definition struct {
	Name        string         `json:"name"`
	Description string         `json:"description"`
	InputSchema map[string]any `json:"input_schema"`
}

type Call struct {
	Name      string          `json:"name"`
	Arguments json.RawMessage `json:"arguments"`
}

type Result struct {
	Tool   string `json:"tool"`
	OK     bool   `json:"ok"`
	Output string `json:"output"`
	Error  string `json:"error,omitempty"`
}

type Runtime struct {
	WorkspaceRoot string
	ForceShell    bool
	In            *os.File
	Out           *os.File
	Err           *os.File
	Delegate      func(ctx context.Context, prompt string) (string, error)
}

func (r Runtime) Definitions() []Definition {
	return []Definition{
		{
			Name:        "read_file",
			Description: "Read a UTF-8 text file from the workspace, optionally by line range.",
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"path":       map[string]any{"type": "string"},
					"start_line": map[string]any{"type": "integer"},
					"end_line":   map[string]any{"type": "integer"},
				},
				"required": []string{"path"},
			},
		},
		{
			Name:        "write_file",
			Description: "Create or overwrite a UTF-8 text file inside the workspace.",
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"path":    map[string]any{"type": "string"},
					"content": map[string]any{"type": "string"},
				},
				"required": []string{"path", "content"},
			},
		},
		{
			Name:        "execute_shell",
			Description: "Execute a shell command in the workspace. Requires interactive confirmation unless force mode is enabled.",
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"command": map[string]any{"type": "string"},
				},
				"required": []string{"command"},
			},
		},
		{
			Name:        "delegate_task",
			Description: "Spawn an isolated sub-agent for a sub-task and return its result.",
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"prompt": map[string]any{"type": "string"},
				},
				"required": []string{"prompt"},
			},
		},
	}
}

func (r Runtime) Execute(ctx context.Context, call Call) Result {
	switch call.Name {
	case "read_file":
		return r.readFile(call.Arguments)
	case "write_file":
		return r.writeFile(call.Arguments)
	case "execute_shell":
		return r.executeShell(ctx, call.Arguments)
	case "delegate_task":
		return r.delegateTask(ctx, call.Arguments)
	default:
		return Result{Tool: call.Name, OK: false, Error: fmt.Sprintf("unknown tool %q", call.Name)}
	}
}

type readFileArgs struct {
	Path      string `json:"path"`
	StartLine int    `json:"start_line"`
	EndLine   int    `json:"end_line"`
}

func (r Runtime) readFile(raw json.RawMessage) Result {
	var args readFileArgs
	if err := json.Unmarshal(raw, &args); err != nil {
		return Result{Tool: "read_file", OK: false, Error: fmt.Sprintf("parse arguments: %v", err)}
	}
	fullPath, err := r.safePath(args.Path)
	if err != nil {
		return Result{Tool: "read_file", OK: false, Error: err.Error()}
	}
	contents, err := os.ReadFile(fullPath)
	if err != nil {
		return Result{Tool: "read_file", OK: false, Error: err.Error()}
	}

	lines := strings.Split(string(contents), "\n")
	start := args.StartLine
	end := args.EndLine
	if start <= 0 {
		start = 1
	}
	if end <= 0 || end > len(lines) {
		end = len(lines)
	}
	if start > end {
		return Result{Tool: "read_file", OK: false, Error: "start_line cannot be greater than end_line"}
	}

	var out strings.Builder
	for i := start - 1; i < end; i++ {
		fmt.Fprintf(&out, "%d | %s\n", i+1, lines[i])
	}
	return Result{Tool: "read_file", OK: true, Output: out.String()}
}

type writeFileArgs struct {
	Path    string `json:"path"`
	Content string `json:"content"`
}

func (r Runtime) writeFile(raw json.RawMessage) Result {
	var args writeFileArgs
	if err := json.Unmarshal(raw, &args); err != nil {
		return Result{Tool: "write_file", OK: false, Error: fmt.Sprintf("parse arguments: %v", err)}
	}
	fullPath, err := r.safePath(args.Path)
	if err != nil {
		return Result{Tool: "write_file", OK: false, Error: err.Error()}
	}
	if err := os.MkdirAll(filepath.Dir(fullPath), 0o755); err != nil {
		return Result{Tool: "write_file", OK: false, Error: err.Error()}
	}
	if err := os.WriteFile(fullPath, []byte(args.Content), 0o644); err != nil {
		return Result{Tool: "write_file", OK: false, Error: err.Error()}
	}
	return Result{Tool: "write_file", OK: true, Output: fmt.Sprintf("wrote %s", args.Path)}
}

type executeShellArgs struct {
	Command string `json:"command"`
}

func (r Runtime) executeShell(ctx context.Context, raw json.RawMessage) Result {
	var args executeShellArgs
	if err := json.Unmarshal(raw, &args); err != nil {
		return Result{Tool: "execute_shell", OK: false, Error: fmt.Sprintf("parse arguments: %v", err)}
	}
	if strings.TrimSpace(args.Command) == "" {
		return Result{Tool: "execute_shell", OK: false, Error: "command is required"}
	}
	if !r.ForceShell && !r.confirmShell(args.Command) {
		return Result{Tool: "execute_shell", OK: false, Error: "command not approved by user"}
	}

	cmd := exec.CommandContext(ctx, "powershell", "-NoProfile", "-Command", args.Command)
	cmd.Dir = r.WorkspaceRoot
	var combined bytes.Buffer
	cmd.Stdout = &combined
	cmd.Stderr = &combined
	if err := cmd.Run(); err != nil {
		return Result{Tool: "execute_shell", OK: false, Output: combined.String(), Error: err.Error()}
	}
	return Result{Tool: "execute_shell", OK: true, Output: combined.String()}
}

type delegateTaskArgs struct {
	Prompt string `json:"prompt"`
}

func (r Runtime) delegateTask(ctx context.Context, raw json.RawMessage) Result {
	if r.Delegate == nil {
		return Result{Tool: "delegate_task", OK: false, Error: "delegate_task is not configured"}
	}
	var args delegateTaskArgs
	if err := json.Unmarshal(raw, &args); err != nil {
		return Result{Tool: "delegate_task", OK: false, Error: fmt.Sprintf("parse arguments: %v", err)}
	}
	if strings.TrimSpace(args.Prompt) == "" {
		return Result{Tool: "delegate_task", OK: false, Error: "prompt is required"}
	}
	output, err := r.Delegate(ctx, args.Prompt)
	if err != nil {
		return Result{Tool: "delegate_task", OK: false, Error: err.Error()}
	}
	return Result{Tool: "delegate_task", OK: true, Output: output}
}

func (r Runtime) confirmShell(command string) bool {
	out := r.Out
	in := r.In
	if out == nil {
		out = os.Stdout
	}
	if in == nil {
		in = os.Stdin
	}
	fmt.Fprintf(out, "Approve shell command? [y/N]\n%s\n> ", command)
	reader := bufio.NewReader(in)
	answer, err := reader.ReadString('\n')
	if err != nil {
		return false
	}
	answer = strings.TrimSpace(strings.ToLower(answer))
	return answer == "y" || answer == "yes"
}

func (r Runtime) safePath(requestPath string) (string, error) {
	if strings.TrimSpace(requestPath) == "" {
		return "", fmt.Errorf("path is required")
	}
	root, err := filepath.Abs(r.WorkspaceRoot)
	if err != nil {
		return "", fmt.Errorf("resolve workspace root: %w", err)
	}
	fullPath := requestPath
	if !filepath.IsAbs(requestPath) {
		fullPath = filepath.Join(root, requestPath)
	}
	fullPath, err = filepath.Abs(fullPath)
	if err != nil {
		return "", fmt.Errorf("resolve path: %w", err)
	}
	rel, err := filepath.Rel(root, fullPath)
	if err != nil {
		return "", fmt.Errorf("check path: %w", err)
	}
	if rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("path %q escapes workspace root", requestPath)
	}
	return fullPath, nil
}
