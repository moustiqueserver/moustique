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
	b64 := base64.StdEncoding.EncodeToString([]byte(s))
	return rotate(b64)
}

func Dec(s string) string {
	if s == "" {
		return ""
	}
	rotated := rotate(s)
	decoded, _ := base64.StdEncoding.DecodeString(rotated)
	return string(decoded)
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
