package oauth

import (
	// "bytes"
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"regexp"
	"strings"

	// "errors"
	"fmt"

	"github.com/bluesky-social/indigo/atproto/identity"
	"github.com/bluesky-social/indigo/atproto/syntax"
)

type PKCE struct {
	Verifier  string
	Challenge string
	method    string
}

func getDidFromHandle(ctx context.Context, handle syntax.Handle) (*identity.Identity, error) {
	userIdentity, err := identity.DefaultDirectory().LookupHandle(ctx, handle)

	return userIdentity, err
}

func GetDocumentFromHandle(ctx context.Context, handle syntax.Handle) (identity.DIDDocument, error) {
	userIdentity, err := getDidFromHandle(ctx, handle)
	if err != nil {
		return identity.DIDDocument{}, err
	}

	userIdentity.PDSEndpoint()

	// signal:
	// {
	// 	scope: 'atproto transition:generic',
	// }

	resolver := identity.BaseDirectory(identity.BaseDirectory{})

	document, err := resolver.ResolveDID(ctx, userIdentity.DID)

	return *document, err
}

func getResourceServeMetadata(pdsUrl string) {
	// fetchMetadata
	// const request = new Request(url, {
	// 	signal: options?.signal,
	// 	headers: { accept: 'application/json' },
	// 	cache: options?.noCache ? 'no-cache' : undefined,
	// 	redirect: 'manual', // response must be 200 OK
	// })
}

func normalizeHandle(handle string) string {
	return strings.ToLower(handle)
}

func EnsureValidHandle(handle string) (bool, error) {
	asciiCharsPattern := `^[a-zA-Z0-9.-]*$`
	handleRegex := regexp.MustCompile(asciiCharsPattern)

	asciiLettersPattern := `^[a-zA-Z]`
	labelRegex := regexp.MustCompile(asciiLettersPattern)

	if !handleRegex.MatchString((handle)) {
		return false, errors.New("handle contains invalid characters")
	}

	if len(handle) > 253 {
		return false, errors.New("handle is too long (253 chars max)")
	}

	labels := strings.Split(handle, ".")
	if len(labels) < 2 {
		return false, errors.New("handle domain needs at least two parts")
	}

	for i := 0; i < len(labels); i++ {
		label := labels[i]

		if len(label) < 1 {
			return false, errors.New("handle parts can not be empty")
		}

		if len(label) > 63 {
			return false, errors.New("handle part too long (max 63 chars)")
		}

		if string(label[0]) == "-" || string(label[len(label)-1]) == "-" {
			return false, errors.New("handle parts can not start or end with hyphens")
		}

		if i+1 == len(labels) && !labelRegex.MatchString(label) {
			return false, errors.New("handle final component (TLD) must start with ASCII letter")
		}
	}

	return true, nil
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

func GenerateDpopKey(algs []string) {

}
