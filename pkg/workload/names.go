package workload

import (
	"math/rand"
	"strings"
)

const (
	consonants = "bcdfghjklmnpqrstvwxz"
	vowels     = "aeiou"
)

func MkKey() string {
	return Name(rnd.Intn(7) + 5)
}

func MkVal() string {
	return Name(rnd.Intn(12) + 4)
}

// Name generates a random string of n characters that only contains consonants and
// vowels. Names are not cryptographically random and are not guaranteed to be unique.
func Name(n int) string {
	if n < 1 {
		return ""
	}

	// Select the consonants to use
	numConsonants := (n / 2) + 1
	chars := make([]byte, 0, n)
	for i := 0; i < numConsonants; i++ {
		chars = append(chars, randConsonant())
	}

	// Select the vowels to use
	numVowels := n - numConsonants
	for i := 0; i < numVowels; i++ {
		chars = append(chars, randVowel())
	}

	// Build the result string
	sb := strings.Builder{}
	sb.Grow(n)
	for remain := len(chars); remain > 0; remain-- {
		idx := rand.Intn(len(chars))
		sb.WriteByte(chars[idx])
		chars = append(chars[:idx], chars[idx+1:]...)
	}

	return sb.String()
}

func randConsonant() byte {
	return consonants[rnd.Intn(len(consonants))]
}

func randVowel() byte {
	return vowels[rnd.Intn(len(vowels))]
}
