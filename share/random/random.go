package random

import (
	"math/rand"
	"time"

	"github.com/google/uuid"
)

const (
	alpha    = "abcdefghijklmnopqrstuvwxyz"
	num      = "0123456789"
	alphaNum = alpha + num
	hex      = num + "abcdef"
)

func init() {
	rand.Seed(time.Now().UnixNano())
}

// AlphaNum returns a random string of length 'n' using alphanumeric characters, regexp: [a-z0-9]{n}.
var AlphaNum = func(n int) string {
	return String(n, alphaNum)
}

// Hex returns a random hex of length 'n', regexp: [a-f0-9]{n}.
func Hex(n int) string {
	return String(n, hex)
}

// Code returns a random string of length 'n' using only numbers, regexp: [0-9]{n}.
func Code(n int) string {
	return String(n, num)
}

// String returns a random string of length 'n' using character set 'chars'.
func String(n int, chars string) string {
	res := make([]byte, n)
	for i := range res {
		res[i] = chars[rand.Intn(len(chars))]
	}
	return string(res)
}

// UUID4 returns a random generated UUID4.
var UUID4 = func() (string, error) {
	uuid, err := uuid.NewRandom()
	if err != nil {
		return "", err
	}
	return uuid.String(), nil
}
