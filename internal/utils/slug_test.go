package utils

import (
	"testing"
)

func TestGenerate(t *testing.T) {
	slug := Generate()
	if len(slug) != SlugLength {
		t.Errorf("expected slug length %d, got %d", SlugLength, len(slug))
	}

	for _, ch := range slug {
		found := false
		for _, a := range SlugAlphabet {
			if ch == a {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("invalid character in slug: %c", ch)
		}
	}
}

func TestValidate(t *testing.T) {
	tests := []struct {
		name    string
		slug    string
		wantErr bool
		errMsg  string
	}{
		{"valid lowercase", "hello", false, ""},
		{"valid mixed", "HelloWorld", false, ""},
		{"valid with dash", "hello-world", false, ""},
		{"valid with underscore", "hello_world", false, ""},
		{"valid numeric", "abc123", false, ""},
		{"empty", "", true, "length"},
		{"too long", string(make([]byte, 65)), true, "length"},
		{"path traversal", "../etc", true, "chars"},
		{"slash", "foo/bar", true, "chars"},
		{"space", "foo bar", true, "chars"},
		{"reserved private", "private", true, "reserved"},
		{"reserved api", "api", true, "reserved"},
		{"reserved install", "install", true, "reserved"},
		{"reserved health", "health", true, "reserved"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := Validate(tt.slug)
			if tt.wantErr {
				if err == nil {
					t.Errorf("Validate(%q) expected error, got nil", tt.slug)
					return
				}
				if inv, ok := err.(*InvalidSlugError); ok && inv.Reason != tt.errMsg {
					t.Errorf("Validate(%q) reason = %q, want %q", tt.slug, inv.Reason, tt.errMsg)
				}
			} else {
				if err != nil {
					t.Errorf("Validate(%q) unexpected error: %v", tt.slug, err)
				}
			}
		})
	}
}

func TestIsReserved(t *testing.T) {
	if !IsReserved("private") {
		t.Error("expected 'private' to be reserved")
	}
	if IsReserved("hello") {
		t.Error("expected 'hello' to not be reserved")
	}
}
