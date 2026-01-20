# iMessage Lightweight CLI Client Project: Implementation Plan

## Project Overview
- **Purpose**: Create a lightweight iMessage client for personal use on Linux.
- **Components**:
  - **Mac Registration Provider**: Generates registration data on a Mac periodically (monthly).
  - **Linux CLI Client**: 
    - Poll for new message updates using a separate CLI command (called by a cron job or scheduled task).
    - Establish a connection to Apple's iMessage servers when launching the client for real-time interaction.

---

## High-Level Architecture

```
iMessageClientProject/
├── mac-registration-provider/     # Tool to run periodically on a Mac (requires macOS/Objective-C)
│   ├── main.go                    # Entry point for generating registration data
│   ├── generate.go                # Validation data generation via NAC
│   ├── nac/                       # Apple NAC (Native Authentication Context) utilities
│   ├── requests/                  # HTTP requests to Apple's identity services
│   └── versions/                  # Device version info
├── imessage-client/               # Core CLI client for Linux
│   ├── cmd/                       # CLI commands
│   │   ├── root.go                # Root command with --registration and --store flags
│   │   ├── check_messages.go     # Command to check for new messages (polling)
│   │   └── send_message.go       # Command to send messages
│   ├── messaging/                 # iMessage handling utilities
│   │   ├── client.go              # High-level client API
│   │   ├── session.go             # Session management with APNS connection
│   │   ├── handshake.go           # Handshaker interface
│   │   ├── handshake_real.go      # Real handshake with key generation
│   │   ├── receive.go             # Message receiving logic
│   │   ├── send.go                # Message sending logic (stub)
│   │   ├── message.go             # Message data structures
│   │   ├── store.go               # Message state storage interface
│   │   ├── store_file.go          # File-backed storage
│   │   ├── errors.go              # Error definitions
│   │   ├── ids/                   # Identity Services types
│   │   │   ├── config.go          # IDS configuration
│   │   │   ├── register.go        # Registration request/response types
│   │   │   └── errors.go          # IDS error types
│   │   └── apns/                  # Apple Push Notification Service
│   │       ├── connection.go      # TLS connection to courier.push.apple.com
│   │       ├── binary.go          # Binary protocol (TLV encoding)
│   │       ├── commands.go        # APNS command structures
│   │       └── topics.go          # APNS topic constants
│   ├── config/                    # Configuration loading
│   │   └── registration.go        # Registration data parsing
│   ├── notifier/                  # CLI output formatting
│   │   └── cli.go
│   └── main.go                    # Entry point
└── docs/                          # Documentation
    ├── session-progress.md        # Detailed implementation status
    ├── migration-plan.md          # What to port from Beeper code
    ├── handshake-plan.md          # IDS/NAC authentication strategy
    └── usage.md                   # CLI usage examples
```

---

## Implementation Plan

### 1. **Mac Registration Provider**
- **Purpose**: Generate iMessage registration data (validation_data) that the Linux client uses to authenticate with Apple servers.
- **How It Works**:
  - Uses Apple's NAC (Native Authentication Context) framework via CGO/Objective-C
  - Interacts with identityservicesd to generate cryptographically signed validation data
  - Outputs JSON containing validation_data, expiry, and device info

- **Requirements**:
  - **Must run on macOS** (uses Objective-C frameworks not available on Linux)
  - Requires Xcode command line tools
  - Go 1.20+ with CGO enabled

- **Steps**:
  1. Stripped Beeper-specific features (relay, submit modes)
  2. Simplified to single `--out` flag for one-time generation
  3. Output format:
     ```json
     {
       "validation_data": "base64_encoded_bytes",
       "valid_until": "2026-02-01T00:00:00Z",
       "nacserv_commit": "commit_hash",
       "device_info": {
         "hardware_version": "MacBookPro18,1",
         "software_name": "macOS",
         "software_version": "13.4.1",
         "software_build_id": "22F82",
         "serial_number": "...",
         "hostname": "..."
       }
     }
     ```
  4. Usage:
     ```bash
     Standalone CLI command to check for new messages
   - Invoked manually or by cron job/systemd timer
   - Loads registration data and state file
   - Connects to APNS, fetches messages, filters against store
   - Outputs unread message summaries

   **Example**:
   ```bash
   $ ./imessage-client check-messages --registration data.json --store state.json
   You have 2 new messages!
   - Alice: "Hey! Are you free for lunch?"
   - Bob: "Here's the file I promised."
   ```

2. **Send Message Command**:
   - Send iMessage to a specific chat/recipient
   - Requires handshake and APNS connection
   
   **Example**:
   ```bash
   $ ./imessage-client send --chat alice@example.com "Hey, how are you?"
   Sent!
   ```

3. **Interactive Mode** (Future):
   - When launched without subcommand, provides interactive interface
   - Real-time message sending/receiving
   - Optional TUI using libraries like `tviewMessage Interaction Command**:
   - When launched, esta (using `spf13/cobra`):
   - `check-messages`: Poll for new unread messages
   - `send`: Send a message to a chat
   - Default (future): Interactive CLI interface

2. **Core Components**:
   - **Messaging Layer**:
     - `Client`: High-level API for poll/send operations
     - `Session`: Manages IDS handshake and APNS connection
     - `Handshaker`: Interface for authentication (RealHandshaker generates keys)
     - `Store`: Tracks last-seen timestamps per chat (file-backed or in-memory)
   
   - **IDS (Identity Services)**:
     - Registration types (RegisterReq, RegisterResp, PrivateDeviceData)
     - Configuration (IDSConfig with keypairs and certificates)
     - Error handling (IDSStatus, IDSError)
   
   - **APNS (Apple Push Notification Service)**:
     - Binary protocol with TLV encoding (Payload, Field, CommandID)
     - TLS connection to courier.push.apple.com:5223
     - Commands: Connect, Filter, SetState, KeepAlive, SendMessage
     - Topic subscription (SHA1 hashing)
     - Message read loop with handler callbacks
   
   - **Message Pipeline**:
     - APNS ReadLoop → handleAPNSMessage → message channel
     - FetchMessages drains channel → filterUnread → updateStore
     - Returns MessageSummary list

3. **Authentication Flow**:
   - Load validation_data from registration JSON
   - Generate IDS keys: RSA 1280 (encryption), ECDSA P256 (signing), RSA 2048 (auth)
   - TODO: Use validation_data to get auth/ID certificates from Apple
   - TODO: Get push certificate and token
   - Connect to APNS with signed nonce
   - Subscribe to com.apple.madrid topic
   - Start background read loopssage-client/
   ├── cmd/
   │   ├── main.go                # Entry point
   │   ├── check-messages.go      # Polling command
   │   ├── send-message.go        # Message sending
   ├── messaging/
   │   ├── sender.go              # Send messages
   │   ├── receiver.go            # Fetch messages
   ├── notifier/
   │   ├── notify.go              # CLI notification output
   ├── config/                    # Load configuration (poll interval, registration path)
   └── ui/                        # Optional terminal UI
   ```

---

## 3. **Interaction Flow**

### a. Workflow: Registration Data Handling
1. Generate registration data on Mac:
   ```bash
   $ ./mac-registration-provider --out registration-data.json
   ```
2. Copy `registration-data.json` to the Linux client.

3. The CLI client loads this file on startup:
   ```json
   {
     "device_id": "device123",
     "auth_tokens": { "access_token": "abc123" }
   }
   ```

---

### b. Polling Command
1. User (or system task) runs:
   ```bash
   $ ./imessage-client check-messages
   ```
2. Flow:
   - Load `registration-data.json`.
   - Authenticate with Apple’s iMessage servers.
   - Check inbox for unread messages.
   - Output summary to terminal or an external notification system.

---

### c. CLI Messaging Interface
1. User launches interactive CLI:
   ```bash
   $ ./imessage-client
   ```
2. Flow:
### Implementation Status (as of Jan 2026)
**Completed (90% MVP):**
- ✅ CLI scaffolding with Cobra
- ✅ Registration data loading and validation
- ✅ File-backed message state storage
- ✅ IDS type system (registration, config, errors)
- ✅ APNS binary protocol (TLV encoding/decoding)
- ✅ APNS TLS connection with authentication
- ✅ Message accumulation and filtering pipeline
- ✅ Handshake with key generation
- ✅ Message decryption (RSA+AES pair encryption, gzip decompression, plist parsing)
- ✅ IDS HTTP client (POST to Apple endpoints, request signing, plist marshaling)
- ✅ IDS registration flow (builds RegisterReq with validation_data, parses response)

**Ready for Testing:**
- 🧪 End-to-end flow with real registration data from Mac
- 🧪 Handshake → IDS register → APNS connect → receive messages
- 🧪 Message decryption with real encrypted payloads

**Future:**
- 🔴 Message encryption for sending
- 🔴 Attachment support (MMCS upload/download)
- 🔴 Group chat handling
- 🔴 Message edits and reactions
- 🔴 Interactive TUI mode

### Key Technologies
- **Language**: Go 1.21+
- **CLI Framework**: `spf13/cobra` v1.8.0
- **UUID Generation**: `github.com/google/uuid` v1.6.0
- **TLS**: Standard library `crypto/tls` with custom ALPN ("apns-security-v3")
- **Binary Protocol**: Custom TLV (Type-Length-Value) encoding
- **Cryptography**:
  - RSA 1280-bit for IDS encryption and push keys
  - ECDSA P256 for IDS signing
  - RSA 2048 for auth keys
  - SHA1 for nonce signing and topic hashing

### Apple Protocol Details
- **APNS Endpoint**: `{1-50}-courier.push.apple.com:5223`
- **Protocol**: Binary TLV over TLS 1.2+
- **Commands**: 0x07 (Connect), 0x08 (ConnectAck), 0x09 (Filter), 0x0a (SendMessage), 0x0c (KeepAlive), 0x14 (SetState)
- **Topics**: com.apple.madrid (main iMessage), com.apple.private.alloy.* (sub-services)
- **Authentication**: Certificate-based with signed nonce (SHA1+RSA)

### Storage
- **State File**: JSON at `${XDG_CONFIG_HOME}/imessage-client/state.json`
- **Format**: `{"chat_id": "2026-01-17T12:00:00Z", ...}`
- **Purpose**: Track last-seen timestamps to filter unread messages

### Security Considerations
- Store `registration-data.json` securely (contains sensitive validation_data)
- Validation data expires (default 15 minutes from generation)
- Need to regenerate monthly or when expired
- All communication with Apple uses TLS
- Message payloads are end-to-end encrypted (TODO: implement decryption)

### Prerequisites
- **For Mac Registration Provider**:
  - macOS 10.15+
  - Xcode command line tools
  - Go 1.20+ with CGO
  
- **For Linux Client**:
  - Linux (any distro)
  - Go 1.21+
  - Valid registration data from Mac provider
   > alice "Hey, lunch works for me!"
   Sent.
   ```

---

## 4. **Technical Notes**

- **Prerequisites**:
  - Install Go 1.20+.
  - Use `nhooyr.io/websocket` for WebSocket communication.
  - Use `spf13/cobra` for CLI parsing.
  - Optional: Use `rivo/tview` for building a TUI.

- **Shared Libraries**:
  - **Encryption**: Extract cryptographic utilities from `beeper/mac-registration-provider`.
  - **Messaging**: Adapt iMessage protocol handling from `beeper/imessage`.

- **Security**:
  - Store `registration-data.json` securely; ensure sensitive tokens don’t leak.
  - Use HTTPS/WebSocket Secure (WSS) for communication with Apple servers.

---

## Example Commands

### Generate Registration Data on Mac:
```bash
$ ./mac-registration-provider --out registration-data.json
```

### Poll for New Messages (cron job/systemd):
```bash
$ ./imessage-client check-messages
```

### Send and Receive Messages:
```bash
$ ./imessage-client
```

--- 

## Future Improvements
- Add support for multimedia messages (images, files).
- Optional message filtering/sorting capability.
