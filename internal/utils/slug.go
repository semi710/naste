package utils

import (
	"crypto/rand"
	"math/big"
	"regexp"
	"strings"
)

const (
	SlugAlphabet = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	SlugLength   = 6
	MaxSlugLen   = 64
	MinSlugLen   = 1
)

var reservedSlugs = map[string]bool{
	"private": true,
	"api":     true,
	"health":  true,
}

var slugRegex = regexp.MustCompile(`^[a-zA-Z0-9_-]+$`)

// Generate creates a random slug of the configured length.
func Generate() string {
	alphabetLen := big.NewInt(int64(len(SlugAlphabet)))
	var sb strings.Builder
	sb.Grow(SlugLength)
	for i := 0; i < SlugLength; i++ {
		n, err := rand.Int(rand.Reader, alphabetLen)
		if err != nil {
			panic(err)
		}
		sb.WriteByte(SlugAlphabet[n.Int64()])
	}
	return sb.String()
}

// Validate checks if a slug is safe and legal.
func Validate(slug string) error {
	if len(slug) < MinSlugLen || len(slug) > MaxSlugLen {
		return &InvalidSlugError{Reason: "length", Message: "slug must be 1-64 characters"}
	}
	if !slugRegex.MatchString(slug) {
		return &InvalidSlugError{Reason: "chars", Message: "slug contains invalid characters"}
	}
	if reservedSlugs[strings.ToLower(slug)] {
		return &InvalidSlugError{Reason: "reserved", Message: "slug is reserved"}
	}
	return nil
}

// InvalidSlugError indicates a slug failed validation.
type InvalidSlugError struct {
	Reason  string
	Message string
}

func (e *InvalidSlugError) Error() string {
	return e.Message
}
