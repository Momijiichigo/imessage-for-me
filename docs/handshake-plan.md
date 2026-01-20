# Handshake & Transport Port Plan

Goal: extract just the Apple-facing pieces from the Beeper bridge (`imessage/imessage/direct`) to power the lightweight CLI. Below maps what to port and what to ignore.

## Key source files to mine
- Message model and payload parsing from [imessage/imessage/struct.go](imessage/imessage/struct.go) (trim Matrix-only fields).
- Transport/auth pipeline in [imessage/imessage/direct/connector.go](imessage/imessage/direct/connector.go) (IDS registration, NAC-serv client usage, APNS handling).
- Payload models and helpers in [imessage/imessage/direct/imessage.go](imessage/imessage/direct/imessage.go) and decrypt/handlers in [imessage/imessage/direct/decrypt.go](imessage/imessage/direct/decrypt.go) (remove analytics and Matrix event wiring).
- IDS primitives under [imessage/imessage/direct/ids](imessage/imessage/direct/ids) (certs, lookup, message encryption) and APNS push handling under `direct/apns`.

## What to strip while porting
- Matrix wiring (event emission, portal/puppet lookups, provisioning) and analytics/Sentry calls sprinkled through `connector.go` and `decrypt.go`.
- Rate limiting and bridge-specific persistence layers; replace with minimal in-memory or small SQLite state for dedup/read state.

## Target shape in `imessage-client/messaging`
- `Session` owns:
  - Parsed registration blob (validation data + device info).
  - NAC-serv client wrapper to derive auth certs.
  - IDS registration lifecycle (refresh/reregister loop simplified or on-demand).
  - APNS connection/handlers to deliver `MessageSummary` items into polling/interactive code.
- `FetchUnread(ctx)` stub will become: ensure IDS registration, drain queued APNS payloads (or a poll equivalent), parse into `MessageSummary`.

## First concrete porting steps
1) Replace `DefaultHandshaker` with `NacIDSHandshaker`: decode `validation_data`, run NAC/IDS auth, and populate `handshakeState` (keys/tokens) used by `Session`.
2) Reproduce minimal `ids.User` state + nacserv client init: strip until it only needs the registration blob and can produce auth/ID certs.
3) Implement a slim APNS receiver that can decrypt payloads using the derived keys (lift from `decrypt.go`, minus analytics and Matrix callbacks).
4) Use the existing message model + store to emit summaries and mark last-seen per chat.

## Safety and footprint
- Remove all AGPL-covered Matrix bridge glue; keep only Apple protocol essentials.
- Replace `zerolog` dependency with stdlib logging where feasible to keep the client lean.
- No background goroutines until a session is explicitly started by a command.
