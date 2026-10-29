package ui

import (
	"fmt"
	"strconv"
	"strings"
)

// validatePort checks that a port string is empty or a valid port number.
func validatePort(s string) error {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil
	}
	p, err := strconv.Atoi(s)
	if err != nil || p <= 0 || p > 65535 {
		return fmt.Errorf("must be a number 1-65535")
	}
	return nil
}

// validateNonNegativeInt checks that a string is empty or a non-negative integer.
func validateNonNegativeInt(s string) error {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil
	}
	n, err := strconv.Atoi(s)
	if err != nil || n < 0 {
		return fmt.Errorf("must be a number >= 0")
	}
	return nil
}
