# Product Requirements Document

## Overview

OmniProxy is a local-first desktop application that acts as an AI API gateway
and multi-account management tool. It runs on the user's machine, proxies AI
API requests through multiple provider accounts, and manages the local
configuration of AI coding clients.

The product exists because users of multiple AI providers and multiple
accounts need one local place to:

- unify access to several AI providers behind a single endpoint,
- distribute load across accounts of the same provider,
- survive quota exhaustion and account failures automatically,
- observe usage, quota and request history,
- configure local AI clients (Codex, Claude Code, Gemini CLI, …) safely.

## Goals

- Provide one local AI API gateway with model routing and account scheduling.
- Support multiple accounts per provider with automatic failover when a
  request fails or an account is rate-limited.
- Track quota, usage and request history per account and provider.
- Manage local client configurations (detect, read, back up, apply, restore,
  validate) without hand-editing config files.
- Run as a Windows desktop application with a system tray, single instance,
  autostart and automatic updates.
- Keep all data local: accounts, credentials, history and settings stay on the
  user's machine.

## Target Users

- Developers and power users with multiple AI provider subscriptions.
- Users of AI coding CLIs who need to switch providers/accounts frequently.
- Anyone who needs visibility into AI API usage and quota consumption.

## Core Use Cases

1. Point an AI client at the local OmniProxy gateway instead of a provider
   endpoint directly.
2. Add a provider account (API key or OAuth credential) once; OmniProxy
   schedules requests across eligible accounts.
3. Route a model name to the provider/account that should serve it.
4. Watch quota and usage; see which account served which request.
5. Detect and configure local AI clients (Codex, Claude Code, Gemini CLI,
   OpenCode) from the UI, with automatic backup and restore.
6. Control the gateway from the desktop UI or from external tools through the
   local Control API.

## Core Features

- Multi-provider access via a unified adapter architecture (OpenAI,
  Anthropic, Google, DeepSeek, Kimi, MiMo, OpenAI-compatible).
- Multi-account management with OS-protected local credential storage; account
  metadata keeps only opaque references to secrets.
- Account scheduling: eligibility, health, priority, load, quota, cooldown,
  concurrency and retry.
- Model routing rules.
- Quota monitoring and usage statistics.
- Request history with success/failure classification.
- Provider health detection.
- Client integration management for external AI tools.
- Local HTTP Control API for CLI/agent/script/MCP access.
- Windows desktop UI, system tray, single instance, autostart, auto-update.

## Non-Goals

Explicitly out of scope for the current stage:

- Cloud sync, multi-user accounts, or any server-side component.
- Mobile applications (iOS/Android build assets exist in the Wails template but
  are not product targets).
- A billing or payment system.
- Prompt management, conversation storage, or any LLM content itself.
  OmniProxy moves requests; it does not store their content beyond metadata.
- Support for provider-specific features beyond the unified adapter interface.
- macOS/Linux desktop support (planned after the Windows product is complete;
  the architecture keeps this possible).
- Replacing or wrapping the provider SDKs beyond request/response translation.

## Constraints

- Local-first: the gateway binds to loopback; no data leaves the machine except
  to the AI providers the user configured.
- Credentials must never be stored in plaintext in the database or logs.
- Development and production environments must not share data directories.
- The Go core must remain independent of Wails and React so it stays testable
  and reusable (e.g. behind a future CLI).

## Success Criteria

- A configured client can complete an AI API request through the local gateway.
- When one account fails or is exhausted, requests are retried on another
  eligible account without user intervention.
- The user can see per-account quota, usage and recent request outcomes.
- Client configurations can be applied and restored from the UI without manual
  file editing.
- The app starts on boot, runs in the tray, and updates itself.

TBD: concrete performance targets, exact quota-refresh semantics per provider,
and the first released version number.
