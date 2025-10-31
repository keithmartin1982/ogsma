package main

import (
	"crypto/rand"
	"log"
	"math/big"
	"strings"
)

func generateRandomString(keyLength int) string {
	chars := []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789_")
	var b strings.Builder
	b.Grow(keyLength)
	for kl := 0; kl < keyLength; kl++ {
		n, err := rand.Int(rand.Reader, big.NewInt(int64(len(chars))))
		if err != nil {
			log.Fatal(err)
		}
		b.WriteRune(chars[n.Int64()])
	}
	return b.String()
}
