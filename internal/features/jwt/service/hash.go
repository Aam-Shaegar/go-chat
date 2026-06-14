package jwt_service

import (
	"crypto/sha256"
	"fmt"
)

func hashToken(token string) string {
	h := sha256.Sum256([]byte(token))
	return fmt.Sprintf("%x", h)
}
