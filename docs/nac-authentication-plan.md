# NAC/IDS Authentication Implementation Plan

## Current Status
We have validation_data from the Mac registration provider, but we're not using it yet for authentication.

## Authentication Flow (Simplified for Personal Use)

### Two Possible Paths:

#### Path 1: Full Authentication (Beeper's approach)
1. Use validation_data to prove device legitimacy  
2. Get auth certificate from Apple (via authenticateDevice endpoint)
3. Use auth cert + IDS keys to register with IDS
4. Get ID certificate back from IDS registration
5. Use push cert/token from registration for APNS

This is complex and requires Apple ID login or SMS verification.

#### Path 2: Direct Registration (Simplified)
1. Generate IDS encryption + signing keys (already doing)
2. Generate push key (already doing)
3. **Use validation_data directly in IDS registration request**
4. Apple validates the validation_data and issues certificates
5. Extract push token and ID cert from response
6. Connect to APNS with push token

**Path 2 is simpler** and should work for our use case since validation_data is generated on a real Mac and is cryptographically signed.

## What Validation Data Contains

From `mac-registration-provider/generate.go`:
```go
func GenerateValidationData(ctx context.Context) ([]byte, time.Time, error) {
    validationCtx, request, err := nac.Init(globalCert)
    // ... talks to Apple's identityservicesd via NAC framework
    // Returns cryptographically signed proof of device legitimacy
}
```

The validation_data is signed by Apple's validation service and contains:
- Device hardware attestation
- Cryptographic proof the device is genuine
- Time-limited validity (15 minutes by default)

## Implementation Steps

### Step 1: Port Register Function
Port the simplified registration flow from Beeper's `register.go`:

```go
func (h *RealHandshaker) Handshake(ctx context.Context, reg *config.RegistrationData) (*handshakeState, error) {
    // 1. Generate IDS keys (already done)
    idsSigningKey, _ := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
    idsEncryptionKey, _ := rsa.GenerateKey(rand.Reader, 1280)
    
    // 2. Generate push key (already done)
    pushKey, _ := rsa.GenerateKey(rand.Reader, 1280)
    
    // 3. Build registration request
    registerReq := ids.RegisterReq{
        DeviceName:        ids.DeviceName,
        HardwareVersion:   reg.DeviceInfo.HardwareVersion,
        OSVersion:         fmt.Sprintf("%s,%s", reg.DeviceInfo.Software Name, reg.DeviceInfo.SoftwareVersion),
        SoftwareVersion:   reg.DeviceInfo.SoftwareBuildID,
        ValidationData:    reg.ValidationData,  // ← KEY: This authenticates us
        Services: []ids.RegisterService{{
            Service: apns.TopicMadrid,
            Users:   []ids.RegisterServiceUser{{
                ClientData: map[string]any{
                    "public-message-identity-key": marshalPublicKey(idsEncryptionKey, idsSigningKey),
                    // ... other capabilities
                },
            }},
        }},
    }
    
    // 4. POST to https://identity.ess.apple.com/.../register
    resp, err := postRegister(ctx, registerReq, pushKey)
    
    // 5. Extract certificates from response
    pushCert := resp.PushCert
    pushToken := resp.PushToken
    idCert := resp.Services[0].Users[0].Cert
    
    // 6. Connect to APNS
    apnsConn := apns.NewConnection(pushKey, pushCert, pushToken)
    
    return &handshakeState{
        IDSConfig: &ids.Config{
            IDSEncryptionKey: idsEncryptionKey,
            IDSSigningKey:    idsSigningKey,
            PushKey:          pushKey,
            PushCert:         pushCert,
            PushToken:        pushToken,
        },
        APNSConn: apnsConn,
    }, nil
}
```

### Step 2: HTTP Request Building
Need to port:
- Request signing with push key
- Proper headers (X-Protocol-Version, User-Agent)
- Plist marshaling/unmarshaling

### Step 3: Response Parsing
Extract from response:
- Push token (for APNS authentication)
- Push certificate (X.509, for APNS TLS)
- ID certificate (for message encryption identity)
- Status codes and error handling

## Key Differences from Beeper

**Beeper approach:**
- Supports full Apple ID login
- Supports SMS registration
- Handles multiple handles (phone + email)
- Needs auth cert renewal
- Complex handle management

**Our approach:**
- No Apple ID login needed
- validation_data is our authentication
- Single handle (whatever Mac used)
- Simpler: just register once per month when validation_data expires

## Files to Create/Modify

1. **`messaging/ids/http.go`** - HTTP client for Apple endpoints
   - POST to registration endpoint
   - Request signing
   - Response parsing

2. **`messaging/handshake_real.go`** - Implement real handshake
   - Build RegisterReq with validation_data
   - Call registration endpoint
   - Parse response
   - Create APNS connection

3. **`messaging/ids/register.go`** - Add response types
   - Already have RegisterReq/RegisterResp
   - Add helper methods

## Testing Strategy

1. **Unit test** registration request building
2. **Integration test** with mock server
3. **End-to-end test** with real validation_data:
   ```bash
   $ cd mac-registration-provider
   $ go build && ./mac-registration-provider --out ../test-data.json
   $ cd ../imessage-client
   $ go build && ./imessage-client check-messages --registration ../test-data.json
   ```

## Expected Behavior

**Success case:**
```
Loading registration data...
Performing IDS handshake...
Registering with IDS...
Got push token: abc123...
Connecting to APNS...
Connected to courier.push.apple.com:5223
Waiting for messages...
```

**Failure cases:**
- Validation data expired → regenerate on Mac
- Invalid validation data → check Mac provider
- Network error → retry with backoff
- Apple returns error status → log details

## Next Steps (in order)

1. ✅ Understand validation_data usage
2. ⏭️ Create HTTP client for IDS endpoints
3. ⏭️ Implement registration request building
4. ⏭️ Implement response parsing
5. ⏭️ Test with real data

## References

- Beeper: `imessage/imessage/direct/ids/register.go`
- Beeper: `imessage/imessage/direct/ids/authdevice.go`
- Mac provider: `mac-registration-provider/generate.go`
- Apple endpoints: `messaging/ids/register.go` (already has URLs)
