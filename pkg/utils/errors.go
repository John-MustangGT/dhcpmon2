// ===== pkg/utils/errors.go =====
package utils

import (
	"fmt"
	"log"
)

// CheckFatal logs a fatal error and exits if err is not nil
func CheckFatal(err error, context string) {
	if err != nil {
		log.Fatalf("%s: %v", context, err)
	}
}

// CheckWarn logs a warning and returns true if err is not nil
func CheckWarn(err error, context string) bool {
	if err != nil {
		log.Printf("Warning - %s: %v", context, err)
		return true
	}
	return false
}

// WrapError wraps an error with additional context
func WrapError(err error, context string) error {
	if err == nil {
		return nil
	}
	return fmt.Errorf("%s: %w", context, err)
}

