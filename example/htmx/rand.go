package htmx

import "math/rand"

var alphabet = []rune("ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789")

func UID() string {
	b := make([]rune, 16)
	for i := range b {
		b[i] = alphabet[rand.Intn(len(alphabet))]
	}
	return string(b)
}
