package rand

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
)

const SessionTokenBytes = 32

func Bytes(n int) ([]byte, error) {
	b := make([]byte, n)

	n, err := rand.Read(b)
	if err != nil {
		return nil, fmt.Errorf("bytes: %w", err)
	}

	return b, nil
}

func String(n int) (string, error) {
	b, err := Bytes(n)
	if err != nil {
		return "", fmt.Errorf("string: %w", err)
	}

	return base64.URLEncoding.EncodeToString(b), nil
}

func SessionToken() (string, error) {
	return String(SessionTokenBytes)
}
