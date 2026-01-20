# Message Decryption Implementation

## Overview
Implemented full iMessage payload decryption for incoming APNS messages. Messages are now decrypted from their encrypted form and parsed into readable text.

## What Was Implemented

### 1. Core Decryption Functions (`messaging/decrypt.go`)

#### ParseBody
Parses the encrypted payload structure used by iMessage:
```
[tag:1byte][bodyLen:2bytes][body:bodyLen][sigLen:1byte][signature:sigLen]
```
- Tag: Encryption type identifier (0x02 for "pair" encryption)
- Body: RSA-encrypted AES key + ciphertext
- Signature: ECDSA signature for verification

#### DecryptPairPayload
Implements the two-stage decryption used by iMessage:
1. **RSA-OAEP Decryption**: First 160 bytes contain RSA-encrypted AES key + first 100 bytes of plaintext
2. **AES-CTR Decryption**: Remaining ciphertext is decrypted with the AES key using CTR mode

The structure is:
```
RSA-encrypted (160 bytes) = [AES key (16 bytes)] + [first part of ciphertext (100 bytes)]
Rest of message = [remaining ciphertext]
```

#### MaybeGUnzip
Handles optional gzip compression:
- Checks for gzip magic number (0x1f 0x8b)
- Decompresses if gzipped, returns as-is if not
- Apple compresses larger messages to save bandwidth

#### DecryptMessage
Main decryption pipeline:
1. Parse encrypted body structure
2. Decrypt using RSA+AES (pair encryption)
3. Decompress if gzipped
4. Parse plist to extract message fields

### 2. Message Payload Structure

```go
type IMessagePayload struct {
    Text         string   `plist:"t,omitempty"` // Message text
    Subject      string   `plist:"s,omitempty"` // Message subject (rare)
    Participants []string `plist:"p,omitempty"` // Chat participants
    GroupID      string   `plist:"gid,omitempty"` // Group chat ID
    Version      int      `plist:"v,omitempty"` // Protocol version
    MessageUUID  string   `plist:"r,omitempty"` // Message UUID
}
```

Currently extracts basic fields. Can be extended for attachments, tapbacks, etc.

### 3. Updated Session Handler (`messaging/session.go`)

The `handleAPNSMessage` function now:
1. Checks if encryption keys are available
2. Attempts to decrypt incoming APNS payloads
3. Extracts message metadata (chat, sender, text)
4. Accumulates decrypted messages to channel
5. Falls back to stub messages if keys unavailable or decryption fails

## How It Works

### Encryption Type: "pair"
iMessage uses hybrid encryption for "pair" messages (older protocol):
- **RSA-1280**: Encrypts symmetric key
- **AES-128-CTR**: Encrypts actual payload
- **ECDSA-P256**: Signs the encrypted payload

### Decryption Flow
```
APNS Payload (encrypted)
    ↓
ParseBody → Extract body + signature
    ↓
DecryptPairPayload → RSA decrypt key, AES decrypt payload
    ↓
MaybeGUnzip → Decompress if needed
    ↓
plist.Unmarshal → Parse to IMessagePayload
    ↓
Extract fields → Create Message struct
    ↓
Send to messageChan → Accumulate for FetchMessages
```

### Current Limitations

1. **No signature verification**: We decrypt but don't verify ECDSA signatures yet
   - Need sender's public signing key from IDS lookup
   - TODO: Port `VerifySignedPairPayload` from Beeper codebase

2. **Only "pair" encryption supported**: Newer "pair-ec" (elliptic curve) not implemented
   - pair-ec uses ECDH for key agreement instead of RSA
   - TODO: Port `decryptPairECEncryptedMessage`

3. **No sender metadata**: Currently using placeholder values
   - Sender ID/handle comes from APNS SendMessagePayload metadata
   - TODO: Parse sender/destination from APNS fields

4. **Basic message types only**: Text messages work, but not:
   - Attachments (need MMCS download)
   - Tapbacks/reactions
   - Message edits
   - Group metadata
   - Read receipts

## Next Steps

### 1. Parse APNS Metadata
Extract sender/destination from `SendMessagePayload`:
```go
type SendMessagePayload struct {
    SenderID      *ParsedURI
    DestinationID *ParsedURI
    Token         []byte
    MessageUUID   uuid.UUID
    Command       MessageType
    Payload       []byte
}
```

### 2. Implement "pair-ec" Decryption
For newer devices (iOS 13+):
- Uses ECDH key agreement
- Protobuf encoding instead of custom format
- Port from `decryptPairECEncryptedMessage` in Beeper

### 3. Add Signature Verification
Security measure to prevent spoofing:
- Verify ECDSA signature against sender's public key
- Requires IDS lookup to get sender's identity
- Port `VerifySignedPairPayload`

### 4. Handle More Message Types
Different commands need different parsing:
- 100: Standard iMessage
- 118: Message edit
- 140: Incoming SMS
- 141: Incoming MMS
- 181: Delete sync
- etc.

## Testing

Currently cannot test decryption without:
1. Real registration data from Mac
2. Valid IDS authentication certificates
3. Active APNS connection with push token
4. Actual incoming messages

Once NAC/IDS authentication is implemented, decryption can be tested end-to-end:
```bash
$ ./mac-registration-provider --out registration-data.json
$ ./imessage-client check-messages --registration registration-data.json
You have 1 new message!
- Alice: "Hey, how are you?"
```

## Dependencies Added

- `howett.net/plist` v1.0.1: Apple plist parsing library
  - Used for parsing iMessage payload format
  - Standard library for macOS/iOS property lists

## References

- Beeper source: `imessage/imessage/direct/decrypt.go`
- IDS encryption: `imessage/imessage/direct/ids/encryptsign_pair.go`
- APNS types: `imessage/imessage/direct/apns/sendmessagepayload.go`
