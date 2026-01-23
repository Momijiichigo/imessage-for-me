package ids

import (
	"fmt"
	"time"

	"github.com/google/uuid"
)

// AppleEpoch is the reference time for Apple timestamps (2001-01-01 00:00 UTC).
var AppleEpoch = time.Date(2001, 1, 1, 0, 0, 0, 0, time.UTC)

// DeviceName is the default device name for registrations.
const DeviceName = "imessage-client"

// ProtocolVersion is the IDS protocol version to use.
const ProtocolVersion = "1640"

// IDS API endpoints
const (
	idsRegisterURL   = "https://identity.ess.apple.com/WebObjects/TDIdentityService.woa/wa/register"
	idsAuthDevURL    = "https://identity.ess.apple.com/WebObjects/TDIdentityService.woa/wa/authenticateDevice"
	idsGetHandlesURL = "https://profile.ess.apple.com/WebObjects/VCProfileService.woa/wa/idsGetHandles"
)

// RegisterReq is the main IDS registration request payload.
type RegisterReq struct {
	DeviceName        string            `plist:"device-name"`
	HardwareVersion   string            `plist:"hardware-version"`
	Language          string            `plist:"language"`
	OSVersion         string            `plist:"os-version"`
	SoftwareVersion   string            `plist:"software-version"`
	PrivateDeviceData PrivateDeviceData `plist:"private-device-data"`
	Services          []RegisterService `plist:"services"`
	ValidationData    []byte            `plist:"validation-data"`
}

// PrivateDeviceData contains device-specific metadata for registration.
type PrivateDeviceData struct {
	AP string `plist:"ap,omitempty"` // "0" on mac, "1" on iphone/ipad
	D  string `plist:"d,omitempty"`  // Seconds (with floating point) since Apple epoch
	DT int    `plist:"dt,omitempty"` // Device type: 1=mac, 2=iphone, 4=ipad
	GT string `plist:"gt,omitempty"` // "0"
	H  string `plist:"h,omitempty"`  // "1"
	M  string `plist:"m,omitempty"`  // "0" on mac/ipad, "1" on iphone
	P  string `plist:"p,omitempty"`  // "0" on mac/ipad, "1" on iphone

	SoftwareBuild   string `plist:"pb,omitempty"` // e.g., "22F82"
	SoftwareName    string `plist:"pn,omitempty"` // e.g., "macOS"
	SoftwareVersion string `plist:"pv,omitempty"` // e.g., "13.4.1"

	S    string `plist:"s,omitempty"` // "0"
	T    string `plist:"t,omitempty"` // "0"
	UUID UUID   `plist:"u,omitempty"` // plist-compatible UUID wrapper
	V    string `plist:"v,omitempty"` // "1", version?
}

// UUID wraps uuid.UUID for plist encoding.
type UUID struct {
	uuid.UUID
}

// RegisterService describes a service being registered (e.g., com.apple.madrid for iMessage).
type RegisterService struct {
	Capabilities []RegisterServiceCapabilities `plist:"capabilities"`
	Service      string                        `plist:"service"` // "com.apple.madrid"
	SubServices  []string                      `plist:"sub-services"`
	Users        []RegisterServiceUser         `plist:"users"`
}

// RegisterServiceUser represents a user/handle being registered.
type RegisterServiceUser struct {
	ClientData map[string]any `plist:"client-data"`
	Tag        string         `plist:"tag,omitempty"` // "SIM" for phone numbers
	URIs       []Handle       `plist:"uris"`
	UserID     string         `plist:"user-id"` // Profile ID (e.g., "P:+1234567890")
}

// Handle represents a registered iMessage handle.
type Handle struct {
	URI ParsedURI `plist:"uri"`
}

// RegisterServiceCapabilities describes what the client supports.
type RegisterServiceCapabilities struct {
	Flags   int    `plist:"flags"`
	Name    string `plist:"name"`    // "Messenger"
	Version int    `plist:"version"` // 1
}

// RegisterResp is the response from IDS registration.
type RegisterResp struct {
	Message       string                `plist:"message"`
	Status        IDSStatus             `plist:"status"`
	Services      []RegisterRespService `plist:"services"`
	RetryInterval int                   `plist:"retry-interval"`
}

// RegisterRespService contains registration response for a service.
type RegisterRespService struct {
	Service string                    `plist:"service"`
	Users   []RegisterRespServiceUser `plist:"users"`
	Status  IDSStatus                 `plist:"status"`
}

// RegisterRespServiceUser contains the ID certificate for a registered user.
type RegisterRespServiceUser struct {
	URIs   []RespHandle       `plist:"uris"`
	UserID string             `plist:"user-id"`
	Cert   []byte             `plist:"cert"` // X.509 certificate bytes
	Status IDSStatus          `plist:"status"`
	Alert  *RegisterRespAlert `plist:"alert"`
}

// RespHandle indicates the status of a registered handle.
type RespHandle struct {
	URI    string    `plist:"uri"`
	Status IDSStatus `plist:"status"`
}

// RegisterRespAlert contains error information if registration fails.
type RegisterRespAlert struct {
	Body   string                  `plist:"body"`
	Button string                  `plist:"button"`
	Title  string                  `plist:"title"`
	Action RegisterRespAlertAction `plist:"action"`
}

// RegisterRespAlertAction describes an action for error alerts.
type RegisterRespAlertAction struct {
	Button string `plist:"button"`
	Type   int    `plist:"type"`
	URL    string `plist:"url"`
}

// DeviceAuthReq is the request to authenticate a device and get auth certificates.
type DeviceAuthReq struct {
	AuthenticationData DeviceAuthData `plist:"authentication-data"`
	CSR                []byte         `plist:"csr"` // Certificate signing request
	RealmUserID        string         `plist:"realm-user-id"`
}

// DeviceAuthData contains credentials for device authentication.
type DeviceAuthData struct {
	// For Apple ID login
	AuthToken string `plist:"auth-token,omitempty"`

	// For SMS/push login
	PushToken  []byte   `plist:"push-token,omitempty"`
	Signatures [][]byte `plist:"sigs,omitempty"`
}

// DeviceAuthResp is the response containing the authentication certificate.
type DeviceAuthResp struct {
	Status int    `plist:"status"`
	Cert   []byte `plist:"cert"` // X.509 certificate bytes
}

// IDSOSVersion returns a formatted OS version string for IDS registration.
func (c *Config) IDSOSVersion() string {
	// Format: "macOS,13.4.1,22F82" or similar
	name := c.SoftwareName
	if name == "" {
		name = "macOS"
	}
	version := c.SoftwareVersion
	if version == "" {
		version = "13.4.1"
	}
	build := c.SoftwareBuildID
	if build == "" {
		build = "22F82"
	}
	return fmt.Sprintf("%s,%s,%s", name, version, build)
}

// CombinedVersion returns the combined software version string.
func (c *Config) CombinedVersion() string {
	return "macOS,13.4.1,22F82"
}
