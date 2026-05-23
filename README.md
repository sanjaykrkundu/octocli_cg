# octocli_cg

A fast CLI agent foundation implemented in Go.

## Step 1 scope

- Cobra-based CLI command routing.
- YAML configuration at `~/.octocli_cg/config.yaml` or `--config <path>`.
- Multiple OpenAI-compatible LLM profiles for local, cloud, and custom APIs.
- Provider abstraction through `internal/llm.Client`.
- Streaming chat completions for OpenAI-compatible `/chat/completions` endpoints.

## Quick start

```powershell
go run . config init
go run . config show
go run . chat "Say hello"
```

The generated sample config contains placeholder API key environment variables only; no real secrets are committed.

## Step 2 scope

- JSON-based ReAct loop for tool-driven reasoning.
- New `agent` command with one-shot mode and interactive REPL mode.
- Tool schema and runtime for `read_file`, `write_file`, and `execute_shell`.
- Tool results are fed back into the loop until a `final_answer` is produced.

### Step 2 examples

```powershell
go run . agent "Read README.md and summarize it"
go run . agent
go run . agent --force "Run git status"
```