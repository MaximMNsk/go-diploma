package sha1hash

import (
	"crypto/sha1"
	"encoding/hex"
)

func Hash(value string) string {
	h := sha1.New()
	h.Write([]byte(value))
	return hex.EncodeToString(h.Sum(nil))
}
