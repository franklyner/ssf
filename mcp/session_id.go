package mcp

import (
	"crypto/rand"
	"encoding/hex"
)

func randomSessionID() string {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		panic("mcp: failed to generate session id: " + err.Error())
	}
	return hex.EncodeToString(b)
}
