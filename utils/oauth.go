package oauth

import (
	// "bytes"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"

	// "errors"
	"fmt"
)

type PKCE struct {
	Verifier  string
	Challenge string
	method    string
}

func GenerateVerifier(byteLength int8) string {
	bytes := make([]byte, byteLength)

	_, err := rand.Read(bytes)
	if err != nil {
		fmt.Println("error:", err)
		return err.Error()
	}

	base64String := base64.RawURLEncoding.EncodeToString((bytes))

	return base64String
}

func GeneratePKCE(byteLength int8) (PKCE, error) {
	verifier := GenerateVerifier(byteLength)

	challengeHash := sha256.Sum256([]byte(verifier))
	challengeBase64 := base64.RawURLEncoding.EncodeToString(challengeHash[:])

	return PKCE{
			verifier,
			challengeBase64,
			"S256",
		},
		nil
}
