package jwt_service

import (
	"crypto/sha256"
	"fmt"
)

func HashToken(token string) string {
	h := sha256.Sum256([]byte(token))
	return fmt.Sprintf("%x", h)
}
