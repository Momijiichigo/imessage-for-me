package messaging

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/rsa"
	"fmt"

	"github.com/google/uuid"

	"imessage-client/config"
	"imessage-client/messaging/apns"
	"imessage-client/messaging/ids"
)

// RealHandshaker implements NAC/IDS handshake using validation data.
type RealHandshaker struct {
	// TODO: add nacserv client when ready
}

func (h RealHandshaker) Handshake(ctx context.Context, reg *config.RegistrationData) (*handshakeState, error) {
	if reg == nil || len(reg.ValidationData) == 0 {
		return nil, ErrInvalidRegistrationData
	}

	// Step 1: Generate IDS keypairs (ECDSA P256 for signing)
	idsSigningKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return nil, fmt.Errorf("failed to generate IDS signing key: %w", err)
	}

	// RSA 1280 for encryption (Apple uses shorter keys for IDS)
	idsEncryptionKey, err := rsa.GenerateKey(rand.Reader, 1280)
	if err != nil {
		return nil, fmt.Errorf("failed to generate IDS encryption key: %w", err)
	}

	// Step 2: Generate push keypairs (RSA 1280)
	pushKey, err := rsa.GenerateKey(rand.Reader, 1280)
	if err != nil {
		return nil, fmt.Errorf("failed to generate push key: %w", err)
	}

	// Step 3: Generate auth private key (RSA 2048)
	authPrivateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return nil, fmt.Errorf("failed to generate auth private key: %w", err)
	}

	// Step 4: Initialize IDS config with device info from registration
	deviceUUID := uuid.New()
	idsConfig := &ids.Config{
		IDSEncryptionKey: idsEncryptionKey,
		IDSSigningKey:    idsSigningKey,
		PushKey:          pushKey,
		AuthPrivateKey:   authPrivateKey,
		AuthIDCertPairs:  make(map[string]*ids.AuthIDCertPair),
		DeviceUUID:       deviceUUID,
	}

	// Extract device info from registration
	idsConfig.HardwareVersion = reg.DeviceInfo.HardwareVersion
	idsConfig.SoftwareVersion = reg.DeviceInfo.SoftwareVersion
	idsConfig.SoftwareName = reg.DeviceInfo.SoftwareName
	idsConfig.SoftwareBuildID = reg.DeviceInfo.SoftwareBuildID

	// Default to macOS if not specified
	if idsConfig.HardwareVersion == "" {
		idsConfig.HardwareVersion = "MacBookPro18,1"
	}
	if idsConfig.SoftwareName == "" {
		idsConfig.SoftwareName = "macOS"
	}
	if idsConfig.SoftwareVersion == "" {
		idsConfig.SoftwareVersion = "13.4.1"
	}
	if idsConfig.SoftwareBuildID == "" {
		idsConfig.SoftwareBuildID = "22F82"
	}

	// Step 5: Register with IDS using validation_data
	httpClient := ids.NewHTTPClient()

	// Build registration request
	registerReq := h.buildRegisterRequest(reg, idsConfig, idsEncryptionKey, idsSigningKey)

	// Send registration request
	registerResp, err := httpClient.Register(ctx, registerReq, pushKey)
	if err != nil {
		return nil, fmt.Errorf("IDS registration failed: %w", err)
	}

	// Step 6: Extract certificates and tokens from response
	if len(registerResp.Services) == 0 {
		return nil, fmt.Errorf("no services in registration response")
	}

	service := registerResp.Services[0]
	if len(service.Users) == 0 {
		return nil, fmt.Errorf("no users in registration response")
	}

	user := service.Users[0]
	if user.Cert == nil {
		return nil, fmt.Errorf("no ID certificate in registration response")
	}

	// Parse ID certificate
	idCert, err := ids.ParseCertificate(user.Cert)
	if err != nil {
		return nil, fmt.Errorf("failed to parse ID certificate: %w", err)
	}

	// Extract push token from response (should be in the connect response or service metadata)
	// For now, we'll use a placeholder since the exact location depends on the response structure
	// TODO: Find where push token is actually returned
	var pushToken []byte
	// In real Apple responses, push token comes from APNS connect, not IDS register
	// We'll connect without it first, then get it from the ConnectAck

	// Store ID certificate in config
	idsConfig.AuthIDCertPairs[user.UserID] = &ids.AuthIDCertPair{
		IDCert: idCert,
	}
	idsConfig.ProfileID = user.UserID

	// Step 7: Create APNS connection with push key
	// Note: Push token will be received during APNS connect handshake
	apnsConn := apns.NewConnection(pushKey, nil, pushToken)

	return &handshakeState{
		ValidationData: reg.ValidationData,
		DeviceInfo:     reg.DeviceInfo,
		IDSConfig:      idsConfig,
		APNSConn:       apnsConn,
	}, nil
}

// buildRegisterRequest constructs the IDS registration request.
func (h RealHandshaker) buildRegisterRequest(
	reg *config.RegistrationData,
	cfg *ids.Config,
	encKey *rsa.PrivateKey,
	signKey *ecdsa.PrivateKey,
) *ids.RegisterReq {
	// Marshal public keys for client data
	encPubKey := &encKey.PublicKey
	signPubKey := &signKey.PublicKey

	// Encode public keys (simplified - full implementation needs proper encoding)
	_ = encPubKey
	_ = signPubKey

	return &ids.RegisterReq{
		DeviceName:      ids.DeviceName,
		HardwareVersion: cfg.HardwareVersion,
		Language:        "en-US",
		OSVersion:       cfg.IDSOSVersion(),
		SoftwareVersion: cfg.SoftwareBuildID,
		PrivateDeviceData: ids.PrivateDeviceData{
			AP:              "0", // Mac
			DT:              1,   // Device type: Mac
			GT:              "0",
			H:               "1",
			M:               "0", // Mac
			P:               "0", // Mac
			SoftwareBuild:   cfg.SoftwareBuildID,
			SoftwareName:    cfg.SoftwareName,
			SoftwareVersion: cfg.SoftwareVersion,
			S:               "0",
			T:               "0",
			UUID:            ids.UUID{UUID: cfg.DeviceUUID},
			V:               "1",
		},
		Services: []ids.RegisterService{{
			Capabilities: []ids.RegisterServiceCapabilities{{
				Flags:   1,
				Name:    "Messenger",
				Version: 1,
			}},
			Service: string(apns.TopicMadrid),
			SubServices: []string{
				// Core iMessage sub-services
				"com.apple.private.alloy.gamecenter.imessage",
				"com.apple.private.alloy.safetymonitor",
				"com.apple.private.alloy.biz",
				"com.apple.private.alloy.sms",
			},
			Users: []ids.RegisterServiceUser{{
				ClientData: map[string]interface{}{
					// Basic capabilities
					"supports-ack-v1":              true,
					"supports-audio-messaging-v2":  true,
					"supports-autoloopvideo-v1":    true,
					"supports-be-v1":               true,
					"supports-ca-v1":               true,
					"supports-fsm-v1":              true,
					"supports-fsm-v2":              true,
					"supports-fsm-v3":              true,
					"supports-inline-attachments":  true,
					"supports-keep-receipts":       true,
					"supports-location-sharing":    true,
					"supports-media-v2":            true,
					"supports-photos-extension-v1": true,
					"supports-st-v1":               true,

					// TODO: Add public key encoding
					// "public-message-identity-key": encodePublicIdentity(encKey, signKey),
					// "public-message-identity-version": 2,
				},
				URIs: []ids.Handle{
					// Will be populated by Apple based on device
				},
				UserID: "", // Will be assigned by Apple
			}},
		}},
		ValidationData: reg.ValidationData,
	}
}
