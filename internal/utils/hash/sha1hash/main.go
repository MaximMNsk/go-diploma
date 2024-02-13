package sha1hash

import (
	"crypto/sha1"
	"encoding/hex"
	"errors"
)

func Hash(value string) (string, error) {
	if len(value) == 0 {
		return ``, errors.New(`empty input item`)
	}
	h := sha1.New()
	h.Write([]byte(value))
	return hex.EncodeToString(h.Sum(nil)), nil
}
