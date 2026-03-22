package reporter

import (
	"log"
	"os"
)

// ReportError centralizes error reporting for unexpected errors.
func ReportError(err error) {
	if err != nil {
		log.Printf("[ERROR] %v", err)
	}
}

// FatalError centralizes fatal error reporting and exits the application.
// It uses %v instead of %w to prevent govet errors since log.Fatalf does not support %w.
func FatalError(format string, err error) {
	if err != nil {
		log.Fatalf("[FATAL] "+format, err)
	} else {
		log.Fatalf("[FATAL] "+format, "")
	}
}
