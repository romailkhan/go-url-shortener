package shortcode

import (
	"crypto/rand"
	"io"
	"math/big"
)

const (
	// DefaultLength is the default short code size (alphanumeric).
	DefaultLength = 8
	alphabet      = "0123456789abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"
)

// Random returns a cryptographically random code of length n using alphabet.
func Random(n int) (string, error) {
	if n < 1 {
		n = DefaultLength
	}
	b := make([]byte, n)
	max := big.NewInt(int64(len(alphabet)))
	for i := range b {
		v, err := rand.Int(Reader, max)
		if err != nil {
			return "", err
		}
		b[i] = alphabet[v.Int64()]
	}
	return string(b), nil
}

// Reader exposes a reader for tests (crypto/rand.Reader in production).
var Reader io.Reader = rand.Reader
