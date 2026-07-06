# AI-agent

`AI-agent` is a Go CLI foundation for an AI coding agent.

## Current Day 1 Status

Day 1 Person 1 work is complete: the project has a basic command router, local initialization, a placeholder chat REPL, and a placeholder task runner. LLM logic, tool calling, file editing, shell execution, and agent reasoning are intentionally left for later branches.

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
go run ./cmd/agent version
go run ./cmd/agent init
go run ./cmd/agent run "summarize this repo"
go run ./cmd/agent chat
```
