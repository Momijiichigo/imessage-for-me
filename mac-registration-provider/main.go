package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/beeper/mac-registration-provider/nac"
	"github.com/beeper/mac-registration-provider/versions"
)

type ReqSubmitValidationData struct {
	ValidationData []byte            `json:"validation_data"`
	ValidUntil     time.Time         `json:"valid_until"`
	NacservCommit  string            `json:"nacserv_commit"`
	DeviceInfo     versions.Versions `json:"device_info"`
}

var Commit = "unknown"

var (
	jsonOutput         = flag.Bool("json", false, "Print JSON to stdout instead of writing a file")
	outputPath         = flag.String("out", "registration-data.json", "Path to write registration data (use - for stdout)")
	checkCompatibility = flag.Bool("check-compatibility", false, "Check if offsets for the current OS version are available and exit")
)

func main() {
	flag.Parse()
	log.Printf("Starting mac-registration-provider %s", shortCommit())
	log.Println("Loading identityservicesd")
	err := nac.Load()
	if err != nil {
		var noOffsetsErr nac.NoOffsetsError
		if errors.As(err, &noOffsetsErr) {
			if *jsonOutput {
				_ = json.NewEncoder(os.Stdout).Encode(map[string]any{
					"error": "no offsets",
					"data":  err,
					"ok":    false,
				})
			}
			log.Fatalf("No offsets found for %s/%s/%s (hash: %s)", noOffsetsErr.Version, noOffsetsErr.BuildID, noOffsetsErr.Arch, noOffsetsErr.Hash)
			return
		}
		panic(err)
	}
	log.Println("Running sanity check...")
	if err = runSanityCheck(); err != nil {
		panic(err)
	}
	if *checkCompatibility {
		log.Println("Compatibility check successful")
		if *jsonOutput {
			_ = json.NewEncoder(os.Stdout).Encode(map[string]any{
				"ok": true,
			})
		}
		return
	}
	log.Println("Fetching certificate...")
	err = InitFetchCert(context.Background())
	if err != nil {
		panic(err)
	}
	log.Println("Generating registration data...")
	validationData, validUntil, err := GenerateValidationData(context.Background())
	if err != nil {
		panic(err)
	}
	payload := &ReqSubmitValidationData{
		ValidationData: validationData,
		ValidUntil:     validUntil,
		NacservCommit:  Commit,
		DeviceInfo:     versions.Current,
	}
	if err := writeOutput(payload); err != nil {
		panic(err)
	}
	log.Println("Registration data ready")
}

func shortCommit() string {
	if len(Commit) >= 8 {
		return Commit[:8]
	}
	return Commit
}

func runSanityCheck() error {
	safetyExitCancel := make(chan struct{})
	defer close(safetyExitCancel)
	go func() {
		select {
		case <-time.After(5 * time.Second):
			log.Fatalln("Sanity check timed out")
		case <-safetyExitCancel:
		}
	}()
	return InitSanityCheck()
}

func writeOutput(payload *ReqSubmitValidationData) error {
	var out *os.File
	var err error
	if *jsonOutput || *outputPath == "-" {
		out = os.Stdout
	} else {
		out, err = os.OpenFile(*outputPath, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o600)
		if err != nil {
			return fmt.Errorf("failed to open output file: %w", err)
		}
		defer out.Close()
	}

	enc := json.NewEncoder(out)
	enc.SetIndent("", "  ")
	if err = enc.Encode(payload); err != nil {
		return fmt.Errorf("failed to encode registration payload: %w", err)
	}
	if out != os.Stdout {
		log.Printf("Wrote registration data to %s", *outputPath)
	}
	return nil
}
