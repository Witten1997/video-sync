# Telegram Bot Integration Review Design

## Context

The current Telegram integration proposal lives in `docs/features/telegram-bot-integration.md`.
The project already has a working URL download flow in `internal/api/handler_video.go`,
download queueing and events in `internal/downloader/manager.go`, config read/write APIs in
`internal/api/handler_config.go`, and a Web config surface in `web/src/views/Config.vue`.

This review focuses on two goals:

1. Make the Telegram integration design safe to implement without forcing later rework.
2. Split the work into stages that can be handed to agents one phase at a time.

The intended first complete milestone includes both the server-side Telegram chain and Web
management support. The milestone is intentionally split into several implementation phases so
agents can deliver it incrementally.

## Recommended Review Stance

Use the existing proposal as the base direction, but revise it around two priorities:

1. Define a real shared application boundary for URL-based submissions.
2. Treat Telegram as a managed product surface, not just a backend adapter.

The current proposal is directionally correct on these points:

- Prefer Long Polling in phase one.
- Do not make Telegram call the existing HTTP API internally.
- Reuse `DownloadManager` and the existing download pipeline.
- Defer group-chat support.

## Required Changes To The Existing Proposal

### 1. Make `URLDownloadService` a real application service

The proposal should stop at neither "move logic out of handler" nor a minimal
`Submit(ctx, req)` signature.

The service contract should explicitly model:

- `trigger_channel`: `web` or `telegram`
- requester identity and display name
- original submitted URL
- request correlation key
- duplicate-submission policy
- result fields needed by both callers

Minimum result contract:

- `video_id`
- `record_id`
- `task_id`
- `source_type`
- `is_existing_video`
- `is_existing_task` or equivalent duplicate outcome
- human-readable title for response text

Without this, Web and Telegram will both reintroduce business rules outside the shared service.

### 2. Split idempotency into adapter idempotency and business idempotency

The current proposal overloads `telegram_request_logs` with too many responsibilities.

The design must separate:

- Telegram adapter idempotency:
  dedupe `update_id`, `chat_id`, `message_id`
- download submission idempotency:
  decide whether the same resolved video should reuse an existing record, reuse an existing task,
  or create a new task

These rules must live at different layers. Telegram replay protection is not the same thing as
download dedupe.

### 3. Define the request-log grain clearly

The current proposal supports multiple URLs in one message but models logging like a single-row
message record.

Phase one should choose one of these designs explicitly:

- only allow one URL per message
- allow multiple URLs, but store one log row per extracted URL

Recommended choice:

- keep multi-URL parsing optional, but store one log row per URL
- keep `message_id` as the grouping key

This removes ambiguity around:

- `status`
- `reply_message_id`
- `task_id`
- `record_id`
- `video_id`

### 4. Define polling offset persistence and commit timing

The proposal says polling should save `offset`, but does not define where or when.

Phase one must define:

- persistence location for the last processed Telegram offset
- whether offset is committed after fetch, after log persistence, or after request handling
- restart recovery behavior
- how duplicate updates are handled after crashes

Long Polling is not production-safe unless this is written down.

### 5. Add first-complete-milestone Web management scope

Because the first complete milestone includes Web management, the proposal must define product
surfaces, not just backend config fields.

Minimum first-complete-milestone Web scope:

- Telegram config section in the existing config UI
- runtime status card:
  enabled, running, mode, last poll time, last error
- Telegram request log page with filters
- linked navigation from Telegram request log to download record

Optional operator actions for later phases:

- test send
- reconnect polling

### 6. Define secret-handling rules

`handleGetConfig` currently returns the in-memory config object, and the Web config UI already
reads configuration directly.

The Telegram proposal must explicitly define:

- `bot_token` is never returned in plaintext to the frontend
- `webhook_secret` is never returned in plaintext to the frontend
- secret update behavior:
  masked value, write-only field, or separate update endpoint
- logging redaction rules

Without this, first-phase Web support creates a direct secret exposure problem.

## Recommended Optimizations

### 1. Separate request history from runtime state

`telegram_request_logs` should not be the only operator view.

Define a lightweight runtime status model, even if it is API-only in phase one:

- `enabled`
- `running`
- `mode`
- `last_poll_at`
- `last_update_id`
- `last_error`
- `last_error_at`

### 2. Prefer editing a single reply message over sending many messages

Telegram notifications should default to:

- send one acceptance message
- edit the same message as the request transitions through queue/completion/failure states
- only send a new message when editing is impossible

This reduces noise and is safer against Telegram rate limits.

### 3. Promote `/status` to an explicit first-phase command

If phase one only sends stage-based notifications and not continuous progress updates, users need
a low-friction status query.

Minimum support:

- `/status` for recent requests by the current user/chat
- `/status <task_id>` for a specific task

### 4. Tighten allowlist and rejection behavior

The proposal should define:

- validation order for chat type, chat ID, and user ID
- fixed rejection messages
- internal rejection reason enums for audit/logging

### 5. Document the limits of in-memory rate limiting

An in-memory limiter is acceptable in early phases, but the proposal should say clearly:

- limits reset on restart
- limits are per-process, not globally consistent across multiple instances
- this is an operational tradeoff, not a permanent guarantee

### 6. Improve external-video identity rules while extracting the shared service

The current URL download path builds a pseudo-BVID for non-Bilibili videos.
The proposal should take this chance to define a more stable external identity model, such as:

- `platform`
- `platform_video_id`
- compatibility mapping to current `bvid` field if migration is deferred

This change is not Telegram-specific, but Telegram will amplify the weakness of the current rule.

### 7. Keep command scope intentionally small

First phase command set should stay minimal:

- `/start`
- `/help`
- `/download <url>`
- direct URL message
- `/status`

Do not turn the bot into a second admin console in the first implementation cycle.

### 8. Define lifecycle listener ownership

If the Telegram module subscribes to `DownloadManager` events, the proposal should define:

- when listeners are registered
- when they are unregistered
- what happens on config reload or Telegram disable/enable transitions

This avoids duplicate notifications after runtime reconfiguration.

## Revised Architecture

```text
Telegram User / Web User
    ->
Entry Adapter
    - Web API handler
    - Telegram update processor
    ->
URLDownloadService
    - normalize request
    - resolve source type
    - apply business idempotency
    - create/reuse video and download record
    - enqueue task
    ->
DownloadManager
    - queue
    - execution
    - events
    ->
Telegram notification + Web visibility
```

Adapter responsibilities:

- input parsing
- identity context
- adapter-level idempotency
- adapter-specific response formatting

Service responsibilities:

- shared URL resolution and submission rules
- video reuse rules
- task creation/reuse rules
- shared submission result contract

Operator surface responsibilities:

- configuration
- runtime visibility
- request history
- cross-linking to download records

## Proposed Delivery Phases

### Phase 1: Extract shared URL download service

Goal:

Make the existing Web URL-download path use a shared application service without changing the
user-facing API.

In scope:

- create `URLDownloadService`
- move Web URL submission logic behind the service
- keep existing `/api/videos/download-by-url`
- preserve current Bilibili and yt-dlp behavior
- define the canonical service request/result contract

Out of scope:

- Telegram config
- Telegram polling
- Telegram logs
- Web Telegram management UI

Acceptance:

- Web URL download still works for Bilibili and non-Bilibili URLs
- handler logic becomes thin
- regression tests cover both submission paths

### Phase 2: Telegram minimal submission chain

Goal:

Deliver private-chat Telegram submission using Long Polling and the shared service.

In scope:

- Telegram config model
- Long Polling client
- offset persistence
- private-chat allowlist
- direct URL and `/download <url>`
- adapter-level idempotency
- basic accept/reject replies

Out of scope:

- Web request-log page
- completion/failure notification workflow
- group chats
- webhook mode

Acceptance:

- an allowed private user can send a URL and create a task
- duplicate updates do not double-submit
- restart does not cause uncontrolled replay

### Phase 3: Request logs and notification loop

Goal:

Make Telegram observable and complete enough for day-to-day use.

In scope:

- `telegram_request_logs`
- one row per extracted URL
- `record_id` as the primary correlation anchor
- completion/failure notification handling
- prefer edit-in-place replies
- `/status` command

Out of scope:

- full Web admin page
- group chats
- webhook mode

Acceptance:

- each accepted request is traceable from Telegram to download record
- completion/failure updates reach the user correctly
- `/status` returns useful results

### Phase 4: Web management support

Goal:

Expose Telegram configuration and operating visibility in the existing Web management surface.

In scope:

- Telegram settings in the config UI
- masked secret handling
- runtime status card
- Telegram request-log page
- request-log filters and links to download records

Out of scope:

- test-send tooling
- force reconnect controls unless they are very low-cost

Acceptance:

- operators can configure Telegram safely from the Web UI
- secrets are not exposed in plaintext
- operators can inspect runtime state and recent request outcomes

### Phase 5: Later enhancements

Goal:

Add non-essential or higher-risk features after the main chain is stable.

Candidate scope:

- webhook mode
- group-chat support
- `@botname` mention handling
- stronger distributed rate limiting
- richer task management commands
- operator actions such as test send or reconnect

## Agent Execution Guidance

For agent-driven delivery, run one phase at a time.

For each phase:

1. restate the exact phase scope
2. write a short implementation plan
3. implement only files owned by that phase
4. verify the phase with focused tests
5. review results before starting the next phase

Avoid combining phases in a single execution request.
The highest-risk boundary is between Phase 1 and Phase 2, so Phase 1 should be accepted before
any Telegram code lands.

## Testing Expectations By Phase

Phase 1:

- unit tests for shared URL submission behavior
- regression tests for the existing HTTP endpoint

Phase 2:

- parser and allowlist tests
- duplicate-update tests
- offset recovery tests

Phase 3:

- request-log persistence tests
- notification routing tests
- `/status` query tests

Phase 4:

- config API tests for masked secrets
- UI smoke coverage for Telegram config and log views

## Final Recommendation

Update `docs/features/telegram-bot-integration.md` to match this review before implementation.

The main design correction is not the Telegram transport itself. It is the need to define a
stable shared URL submission service and a first-phase operator surface.

The recommended delivery target for the first complete milestone is:

- Phase 1
- Phase 2
- Phase 3
- Phase 4

Phase 5 should remain explicitly out of the first implementation cycle.
