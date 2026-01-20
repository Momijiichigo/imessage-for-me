package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"time"
)

// RegistrationData mirrors the output of mac-registration-provider.
type RegistrationData struct {
	ValidationData []byte     `json:"validation_data"`
	ValidUntil     time.Time  `json:"valid_until"`
	NacservCommit  string     `json:"nacserv_commit"`
	DeviceInfo     DeviceInfo `json:"device_info"`
}

type DeviceInfo struct {
	HardwareVersion string `json:"hardware_version"`
	SoftwareName    string `json:"software_name"`
	SoftwareVersion string `json:"software_version"`
	SoftwareBuildID string `json:"software_build_id"`
	SerialNumber    string `json:"serial_number"`
	UniqueDeviceID  string `json:"unique_device_id,omitempty"`
	Hostname        string `json:"hostname"`
}

var ErrMissingRegistration = errors.New("registration data not found")

func LoadRegistration(path string) (*RegistrationData, error) {
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return nil, fmt.Errorf("%w: %s", ErrMissingRegistration, path)
	} else if err != nil {
		return nil, fmt.Errorf("failed to read registration data: %w", err)
	}

	var reg RegistrationData
	if err = json.Unmarshal(data, &reg); err != nil {
		return nil, fmt.Errorf("failed to parse registration data: %w", err)
	}
	return &reg, nil
}

// IsExpired reports whether the validation data is no longer fresh enough to use.
func (r *RegistrationData) IsExpired() bool {
	if r == nil {
		return true
	}
	return time.Now().After(r.ValidUntil)
}
