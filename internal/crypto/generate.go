package crypto

import (
	"crypto/rand"
	"encoding/hex"

	"github.com/pkg/errors"
)

func RandomBytes(n int) ([]byte, error) {
	b := make([]byte, n)
	_, err := rand.Read(b)
	if err != nil {
		return nil, errors.WithStack(err)
	}
	return b, nil
}

func RandomToken(n int) (string, error) {
	bytes, err := RandomBytes(n)
	if err != nil {
		return "", errors.WithStack(err)
	}
	return hex.EncodeToString(bytes), nil
}
