package moustique

import (
	"encoding/base64"
	"strings"
	"time"
)

func Enc(s string) string {
	if s == "" {
		return ""
	}
	// Must match server encoding: ROT13 first, then Base64
	rotated := rotate(s)
	return base64.StdEncoding.EncodeToString([]byte(rotated))
}

func Dec(s string) string {
	if s == "" {
		return ""
	}
	// Reverse of encode: Base64 decode first, then ROT13
	decoded, _ := base64.StdEncoding.DecodeString(s)
	return rotate(string(decoded))
}

func rotate(s string) string {
	from := "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz"
	to := "NOPQRSTUVWXYZABCDEFGHIJKLMnopqrstuvwxyzabcdefghijklm"
	var b strings.Builder
	for _, c := range s {
		idx := strings.IndexRune(from, c)
		if idx != -1 {
			b.WriteRune(rune(to[idx]))
		} else {
			b.WriteRune(c)
		}
	}
	return b.String()
}

func NiceDateTime() string {
	return time.Now().Format("2006-01-02 15:04:05")
}
