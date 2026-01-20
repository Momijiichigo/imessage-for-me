# Project Status & Next Steps

## Completed (January 2026)

### Infrastructure
- ✅ Removed nested .git directories from cloned Beeper repos
- ✅ Created `imessage-client/` skeleton with Cobra CLI framework
- ✅ Created `mac-registration-provider/` simplified to `--out` flag only
- ✅ Set up pluggable storage (in-memory + file-backed JSON)
- ✅ Added configuration loading with expiry checks
- ✅ Implemented CLI commands: `check-messages`, `send` (both stubbed)
- ✅ Added handshaker interface with default stub implementation

### Documentation
- ✅ Migration plan mapping Beeper code to our structure
- ✅ Handshake plan outlining IDS/NAC porting strategy
- ✅ Usage docs and README

## Current State
All CLI commands are functional but return "not implemented" after validating registration data. The handshake succeeds with validation data present but doesn't actually perform IDS/NAC authentication.

## Next Implementation Phase

### Phase 1: Core Handshake (Priority)
1. Port minimal IDS types/config from `imessage/imessage/direct/ids/`
2. Implement `NacIDSHandshaker` using validation data
3. Add APNS connection setup (no full transport yet)
4. Cache derived keys/certs in handshakeState

### Phase 2: Message Receive
1. Port message models from `imessage/imessage/struct.go`
2. Implement APNS payload decryption
3. Wire `FetchUnread` to parse received messages
4. Update store per chat after fetching

### Phase 3: Message Send
1. Port send logic from `imessage/imessage/direct/connector.go`
2. Implement message encryption
3. Wire `Send` to use APNS transport

### Phase 4: Polish
1. Add attachment support
2. Improve error messages
3. Add logging
4. Handle edge cases (reconnection, cert refresh, etc.)

## Key Files to Port

### From `imessage/imessage/direct/ids/`
- `config.go` - IDS configuration structure
- `types/` - Error types and status codes
- `bag.go` - Apple bag configuration
- `register.go` - IDS registration logic
- `encryptsign.go` - Message encryption/signing

### From `imessage/imessage/direct/apns/`
- `connection.go` - APNS connection management
- `payload.go` - Payload structures
- Message type constants

### From `imessage/imessage/direct/`
- `decrypt.go` - Payload decryption logic
- `connector.go` - High-level message handling

## Blockers & Dependencies
- Need to understand which parts of `validation_data` map to what
- APNS connection requires Apple push certs (may be in validation data)
- Some crypto operations may require platform-specific code (minimal since we have validation data)

## Testing Strategy
1. Test registration data loading/validation
2. Mock APNS responses for decrypt testing
3. Integration test with real registration data once handshake works
4. Manual testing with `check-messages` and `send` commands
