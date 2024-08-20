package app

import (
	crypto "crypto/rand"
	"log"
	"math/rand"
)

var secret []byte

func init() {
	secret = make([]byte, 32)
	if _, err := crypto.Read(secret); err != nil {
		log.Fatal("error generating application secret:", err)
	}
}

const letters, numbers = "abcdefghijklmnopqrstuvwxyz", "1234567890"
const idCharsLen, idNumsLen = 4, 3
const idLen = idCharsLen + idNumsLen

func generateId() string {
	l := make([]byte, idCharsLen)
	for i := range l {
		l[i] = letters[rand.Intn(len(letters))]
	}
	n := make([]byte, idNumsLen)
	for i := range n {
		n[i] = numbers[rand.Intn(len(numbers))]
	}
	return string(l) + string(n)
}
