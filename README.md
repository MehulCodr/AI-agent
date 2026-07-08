# AI-agent

`AI-agent` is a Go CLI coding agent with a tool-calling loop, real LLM providers, session memory, Redis-backed repository search, streaming output, and approval-gated shell execution.

## Commands

```sh
go run ./cmd version
go run ./cmd init
go run ./cmd config
go run ./cmd tools
go run ./cmd index
go run ./cmd run "summarize this repo"
go run ./cmd chat
```

Common flags:

```sh
--provider openai|gemini|anthropic|ollama
--model model-name
--session session-name
--no-stream
```

## Configuration

Run `go run ./cmd init` to create `.agent/config.json` and fill missing `.env` keys.

Required `.env` keys:

```sh
AI_AGENT_PROVIDER=gemini
AI_AGENT_MODEL=
AI_AGENT_SESSION=default
AI_AGENT_STREAM=true
AI_AGENT_MAX_STEPS=8

OPENAI_API_KEY=
OPENAI_BASE_URL=https://api.openai.com/v1
GEMINI_API_KEY=
GEMINI_BASE_URL=https://generativelanguage.googleapis.com/v1beta
ANTHROPIC_API_KEY=
ANTHROPIC_BASE_URL=https://api.anthropic.com/v1
OLLAMA_BASE_URL=http://localhost:11434

REDIS_ADDR=localhost:6379
REDIS_PASSWORD=
REDIS_DB=0
```

Set `AI_AGENT_PROVIDER`, `AI_AGENT_MODEL`, and the key for the selected provider. Ollama does not require an API key, but it does require a running Ollama server and a local model.

## Tools

The agent exposes:

- `read_file`, `write_file`, `edit_file`, and `list_files` with project-root path checks.
- `shell`, which asks for user approval before every command.
- `search_repo`, which searches the Redis repository index.
- `current_directory` and `echo` utility tools.

## Repository RAG

Start Redis, then run:

```sh
go run ./cmd index
```

The index stores repository text files in Redis under a project-specific namespace. The `search_repo` tool uses that Redis index during agent runs.

## Verification

```sh
go fmt ./...
go vet ./...
go test ./...
```
