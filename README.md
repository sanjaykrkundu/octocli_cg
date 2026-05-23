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