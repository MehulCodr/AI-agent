# AI-agent

`AI-agent` is a Go CLI foundation for an AI coding agent.

## Current Status

Day 1 CLI and LLM foundations are in place. Day 2 adds the safe tool system foundation with a registry plus basic echo, current directory, and list files tools. LLM tool calling, file editing, shell execution, and agent reasoning are intentionally left for later branches. Day 3 adds project context scanning.

## Available Commands

```sh
agent version
agent init
agent context
agent chat
agent run "task"
```

## How To Run Locally

From the repository root:

```sh
go run ./cmd version
go run ./cmd init
go run ./cmd context
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
