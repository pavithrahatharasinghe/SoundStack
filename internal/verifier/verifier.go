package verifier

import (
	"fmt"
	"os"
	"os/exec"
)

// CheckBinary verifies ffprobe binary availability.
func CheckBinary(path string) error {
	if path == "" {
		return fmt.Errorf("ffprobe path not configured")
	}
	if _, err := os.Stat(path); err == nil {
		return nil
	}
	_, err := exec.LookPath(path)
	return err
}
