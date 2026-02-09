package main

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"regexp"
	"strings"
	"time"
)

func generateName() string {
	stamp := time.Now().Format("20060102-150405")
	buf := make([]byte, 2)
	if _, err := rand.Read(buf); err != nil {
		return "sb-" + stamp
	}
	return "sb-" + stamp + "-" + hex.EncodeToString(buf)
}

func shortHash(input string) string {
	sum := sha256.Sum256([]byte(input))
	return hex.EncodeToString(sum[:6])
}

func containerName(name string) string {
	return "codex-sb-" + normalizeName(name)
}

var validNamePattern = regexp.MustCompile(`^[A-Za-z0-9._-]+$`)

func validName(name string) bool {
	return name != "" && validNamePattern.MatchString(name)
}

func normalizeName(name string) string {
	if name == "" {
		return ""
	}
	var b strings.Builder
	b.Grow(len(name))
	for _, r := range strings.ToLower(name) {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '.' || r == '_' || r == '-' {
			b.WriteRune(r)
			continue
		}
		if r == ' ' || r == '/' || r == '\\' {
			b.WriteByte('-')
		}
	}
	return strings.Trim(b.String(), "-.")
}
