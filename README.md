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

## Step 3 scope

- Rules are loaded from both `~/.octocli_cg/rules/` and `./.agents/rules/`.
- Workflow templates are loaded from both `~/.octocli_cg/workflows/` and `./.agents/workflows/`.
- Workspace workflows override global workflows when names collide.
- Loaded rules are appended to the agent system prompt.
- Slash workflows like `/review` are expanded before the agent loop runs.

### Step 3 examples

```powershell
go run . workflow list
go run . workflow show review
go run . agent /review
go run . agent "/review src/auth.go"
```

## Step 4 scope

- Task artifacts are stored in `./.agents/brain/<GUID>/`.
- Each task contains `task.md`, `implementation_plan.md`, and `.metadata.json`.
- Metadata tracks goal, timestamps, status, and checklist completion.
- New `task` command supports create, list, show, and checklist updates.

### Step 4 examples

```powershell
go run . task create "Refactor auth module" --item "Inspect current code" --item "Write plan" --item "Implement changes"
go run . task list
go run . task show <task-id>
go run . task check <task-id> 0 true
```

## Step 5 scope

- Bubble Tea TUI for monitoring tracked tasks and logs.
- Async/background task runner that updates `.agents/brain/<GUID>/` state.
- `task run` starts a background workflow and immediately returns a task ID.
- `task monitor` opens a live terminal UI for status/log inspection.

### Step 5 examples

```powershell
go run . task run "Refactor auth module" --item "Inspect current code" --item "Write tests" --item "Implement changes"
go run . task monitor
```

## Step 6 scope

- Added `delegate_task` to the agent toolset.
- The main agent can spawn isolated sub-agents for focused subtasks.
- Sub-agents inherit the same workspace capabilities and confirmation rules.
- Delegation depth is bounded to avoid runaway recursive spawning.

### Step 6 examples

```powershell
go run . agent "Use delegate_task to inspect README.md and summarize the implementation roadmap"
```