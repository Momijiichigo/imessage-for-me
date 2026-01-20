package ids

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha1"
	"crypto/x509"
	"encoding/base64"
	"fmt"
	"io"
	"net/http"
	"time"

	"howett.net/plist"
)

// HTTPClient wraps HTTP operations for IDS endpoints.
type HTTPClient struct {
	client *http.Client
}

// NewHTTPClient creates a new IDS HTTP client.
func NewHTTPClient() *HTTPClient {
	return &HTTPClient{
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// Register sends a registration request to Apple's IDS service.
// Returns the parsed response containing push token and certificates.
func (c *HTTPClient) Register(ctx context.Context, req *RegisterReq, pushKey *rsa.PrivateKey) (*RegisterResp, error) {
	// Marshal request to plist
	body, err := plist.Marshal(req, plist.XMLFormat)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal register request: %w", err)
	}

	// Create HTTP request
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, idsRegisterURL, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create HTTP request: %w", err)
	}

	// Set headers
	httpReq.Header.Set("Content-Type", "application/x-apple-plist")
	httpReq.Header.Set("X-Protocol-Version", ProtocolVersion)
	httpReq.Header.Set("User-Agent", fmt.Sprintf("com.apple.invitation-registration [%s]", req.SoftwareVersion))

	// Sign request with push key
	if err := c.signRequest(httpReq, body, pushKey); err != nil {
		return nil, fmt.Errorf("failed to sign request: %w", err)
	}

	// Send request
	resp, err := c.client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to send register request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("register request failed with status %d: %s", resp.StatusCode, string(bodyBytes))
	}

	// Parse response
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	var registerResp RegisterResp
	if _, err := plist.Unmarshal(respBody, &registerResp); err != nil {
		return nil, fmt.Errorf("failed to unmarshal register response: %w", err)
	}

	// Check response status
	if registerResp.Status != 0 {
		return nil, fmt.Errorf("registration failed with status %d: %s", registerResp.Status, registerResp.Message)
	}

	return &registerResp, nil
}

// signRequest signs an HTTP request with the push key for authentication.
func (c *HTTPClient) signRequest(req *http.Request, body []byte, pushKey *rsa.PrivateKey) error {
	// Create signing payload: method + URL + body
	signingPayload := createSigningPayload(req.Method, req.URL.String(), body)

	// Sign with SHA1+RSA
	hashed := sha1.Sum(signingPayload)
	signature, err := rsa.SignPKCS1v15(rand.Reader, pushKey, 0, hashed[:])
	if err != nil {
		return fmt.Errorf("failed to sign payload: %w", err)
	}

	// Add signature header
	req.Header.Set("X-Push-Sig", base64.StdEncoding.EncodeToString(signature))

	// Add push token header (empty for initial registration)
	req.Header.Set("X-Push-Token", "")

	return nil
}

// createSigningPayload creates the payload to sign for request authentication.
func createSigningPayload(method, url string, body []byte) []byte {
	var buf bytes.Buffer
	buf.WriteString(method)
	buf.WriteString("\n")
	buf.WriteString(url)
	buf.WriteString("\n")
	buf.Write(body)
	return buf.Bytes()
}

// AuthenticateDevice requests an auth certificate from Apple (used for Apple ID login).
// For our use case with validation_data, we can skip this and register directly.
func (c *HTTPClient) AuthenticateDevice(ctx context.Context, req *DeviceAuthReq) (*DeviceAuthResp, error) {
	// Marshal request
	body, err := plist.Marshal(req, plist.XMLFormat)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal auth request: %w", err)
	}

	// Create HTTP request
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, idsAuthDevURL, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create HTTP request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/x-apple-plist")
	httpReq.Header.Set("X-Protocol-Version", ProtocolVersion)
	httpReq.Header.Set("User-Agent", "imessage-client")

	// Send request
	resp, err := c.client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to send auth request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("auth request failed with status %d", resp.StatusCode)
	}

	// Parse response
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	var authResp DeviceAuthResp
	if _, err := plist.Unmarshal(respBody, &authResp); err != nil {
		return nil, fmt.Errorf("failed to unmarshal auth response: %w", err)
	}

	if authResp.Status != 0 {
		return nil, fmt.Errorf("authentication failed with status %d", authResp.Status)
	}

	return &authResp, nil
}

// ParseCertificate parses an X.509 certificate from DER bytes.
func ParseCertificate(der []byte) (*x509.Certificate, error) {
	return x509.ParseCertificate(der)
}
