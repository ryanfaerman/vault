package vault

import (
	"crypto/rand"
	"encoding/hex"
)

var TokenSize = 8

func Token() string {
	rb := make([]byte, TokenSize)
	_, err := rand.Read(rb)

	if err != nil {
		panic(err)
	}

	return hex.EncodeToString(rb)
}
