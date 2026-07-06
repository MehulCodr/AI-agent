# AI-agent

`AI-agent` is a Go CLI foundation for an AI coding agent.

## Current Status

Day 1 CLI and LLM foundations are in place. Day 2 adds the core agent loop, mock LLM-backed CLI runs, and the safe tool system foundation with a registry plus basic echo, current directory, and list files tools. Real tool execution, file editing, shell execution, and API-backed providers are intentionally left for later branches.

## Available Commands

```sh
agent version
agent init
agent chat
agent run "task"
```

## How To Run Locally

From the repository root:

```sh
go run ./cmd version
go run ./cmd init
go run ./cmd run "summarize this repo"
go run ./cmd chat
```

## LLM Configuration

Create a local `.env` file to use the Gemini provider:

```sh
GEMINI_API_KEY=your-api-key
GEMINI_MODEL=gemini-3.5-flash
# Optional:
# GEMINI_BASE_URL=https://generativelanguage.googleapis.com/v1beta
```

If `GEMINI_API_KEY` is missing, the CLI uses the local mock provider.
