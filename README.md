# AI-agent

AI-agent is a small Go CLI foundation for an AI coding agent. It is intentionally
CLI-first, readable, and conservative: the current agent can run one-off prompts,
start an interactive chat, scan project context, and use a mock provider for local
demo flows without requiring a real API key.

## Project Overview

The repository is organized around a few focused packages:

- `cmd` starts the CLI.
- `internal/cli` parses commands, loads `.env`, and runs the REPL.
- `internal/agent` manages conversation memory and sessions.
- `internal/llm` provides mock and Gemini provider implementations.
- `internal/context` scans repository context.
- `internal/tools` contains the safe tool registry and basic file/shell tools.

The default provider is `mock`, so the demo commands are safe to run locally. A
real Gemini provider can be enabled with environment variables when needed.

## Installation

Prerequisites:

- Go 1.22 or newer.
- A terminal from the repository root.

Run the CLI directly during development:

```sh
go run ./cmd version
```

Or build a local binary:

```sh
go build -o agent ./cmd
./agent version
```

## API Key Setup

For the local mock provider, copy the example env file and leave the API key
empty:

```sh
cp .env.example .env
```

`.env.example` contains:

```sh
AI_AGENT_PROVIDER=mock
AI_AGENT_MODEL=mock
AI_AGENT_API_KEY=
```

To use Gemini instead, update `.env`:

```sh
AI_AGENT_PROVIDER=gemini
AI_AGENT_MODEL=gemini-3.5-flash
AI_AGENT_API_KEY=your-gemini-api-key
```

The fallback names `GEMINI_API_KEY`, `GEMINI_MODEL`, and `GEMINI_BASE_URL` are
also supported.

## Available Commands

```sh
agent version
agent init
agent context
agent run "task"
agent chat
agent help
```

- `version` prints the CLI version.
- `init` creates `.agent/config.json` if it does not already exist.
- `context` prints a lightweight repository summary.
- `run "task"` sends a single prompt to the configured provider.
- `chat` starts an interactive REPL.
- `help` prints command usage.

Inside `chat`, use `/history`, `/clear`, and `/exit`.

## Example Usage

Initialize project-local agent metadata:

```sh
go run ./cmd/agent init
```

The sample `.agent/config.example.json` mirrors the project metadata created by
`init`; provider selection is controlled by `.env`.

Run a one-off task with the mock provider:

```sh
go run ./cmd/agent run "summarize this repo"
```

Inspect repository context:

```sh
go run ./cmd/agent context
```

Start an interactive chat session:

```sh
go run ./cmd/agent chat
```

## Safety Notes

- The mock provider is the safest default for demos and tests.
- Keep real API keys in `.env`; do not commit `.env`.
- `init` leaves an existing `.agent/config.json` unchanged.
- Shell and file tools are registered for the agent foundation, but model tool
  calling is not enabled yet.
- Review any future edit or shell-execution features before enabling them with a
  real provider.

## Development Workflow

Start from an up-to-date `main` branch:

```sh
git checkout main
git pull origin main
git checkout -b feature/docs-demo
```

Before committing changes:

```sh
go fmt ./...
go test ./...
```

## Demo Script

Run this from the repository root:

```sh
go run ./cmd/agent version
go run ./cmd/agent init
go run ./cmd/agent run "summarize this repo"
go run ./cmd/agent chat
go test ./...
```

Type `/exit` in `chat` to return to the shell before running the test command.
