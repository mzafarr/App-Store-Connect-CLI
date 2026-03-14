package googleplay

import (
	"fmt"
	"strings"

	"github.com/golang-jwt/jwt/v5"
)

// ValidateServiceAccount performs local validation of credential material.
func ValidateServiceAccount(account ServiceAccount) error {
	if strings.TrimSpace(account.ClientEmail) == "" {
		return fmt.Errorf("client_email is required")
	}
	if strings.TrimSpace(account.PrivateKey) == "" {
		return fmt.Errorf("private_key is required")
	}
	if _, err := jwt.ParseRSAPrivateKeyFromPEM([]byte(account.PrivateKey)); err != nil {
		return fmt.Errorf("private_key is invalid: %w", err)
	}
	return nil
}
