package reporter

import (
	"log"
)

// ReportError logs an error centrally with optional metadata.
// This is the required mechanism for handling all unexpected errors
// across the application.
func ReportError(err error, metadata map[string]any) {
	if metadata == nil {
		log.Printf("ERROR: %v", err)
	} else {
		log.Printf("ERROR: %v | Metadata: %v", err, metadata)
	}
}
