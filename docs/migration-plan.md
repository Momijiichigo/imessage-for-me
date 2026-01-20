# Migration Plan (Beeper repos → lightweight CLI)

## What to port first
- Core message model and IPC helpers from `imessage/imessage` (e.g. `struct.go`, `direct/`, `ipc/` as needed for Apple session establishment and message serialization).
- Attachment and conversion helpers from `imessage/msgconv` (image/HEIF/TIFF/url preview parsing) when we enable media handling.
- Minimal local state helpers from `imessage/database` (only pieces needed for tracking message IDs/thread roots; skip Matrix-specific KV/portal/puppet records).
- Crypto and validation path from `mac-registration-provider/nac` and `mac-registration-provider/requests` to keep registration generation working.
- Version metadata from `mac-registration-provider/versions` to record OS/build info alongside registration output.

## What to drop or rewrite
- All Matrix/mautrix glue in `imessage` (`config/bridge.go`, `user.go`, `findrooms.go`, `imessage/ipc/global.go`, etc.).
- Android/puppet/portal management in `imessage/android-*`, `imessage/puppet.go`, and bridge provisioning APIs.
- Analytics/telemetry and Beeper-specific provisioning (`imessage/analytics`, provisioning API logic, Sentry handlers in Barcelona Swift code).
- Registration relay/submission modes in `mac-registration-provider` (`relay.go`, `submit.go`, associated flags). Replace with a single local `--out` flow.

## Target layout (new client)
- `imessage-client/`: CLI entrypoint (`cmd/` with `check-messages` + interactive default), `messaging/` (session handling, send/recv), `ipc/` (encoding/decoding borrowed from `imessage/imessage`), `config/` (registration parsing), `notifier/` (CLI-friendly output). Shared utilities can move here as they are ported.

## Next porting passes
- Map `imessage/imessage/direct` dependencies to remove mautrix imports; replace Matrix event emitters with local message pipeline feeding `messaging`.
- Extract a minimal persistence layer (SQLite or flat-file) for deduplication/read-state without Matrix portal tables.
- Once registration output format is finalized, wire Linux client to read it and initiate Apple session handshake.
