# AI-agent

`AI-agent` is a Go CLI foundation for an AI coding agent.

## Current Status

Day 1 CLI and LLM foundations are in place. Day 2 adds the safe tool system foundation with a registry plus basic echo, current directory, and list files tools. LLM tool calling, file editing, shell execution, and agent reasoning are intentionally left for later branches.

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
