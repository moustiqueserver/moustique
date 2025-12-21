package main

import (
	"crypto/md5"
	"encoding/base64"
	"encoding/hex"
	"io"
	"os"
	"strings"
)

// decodeROT13Base64 decodes the ROT13+Base64 encoding used by moustique_client
func decodeROT13Base64(encoded string) string {
	if encoded == "" {
		return ""
	}
	// Remove whitespace (newlines, etc)
	encoded = strings.TrimSpace(encoded)

	// Apply ROT13
	rotated := rot13(encoded)

	// Decode Base64
	decoded, err := base64.StdEncoding.DecodeString(rotated)
	if err != nil {
		// If it fails, try without ROT13 (might be plain text)
		decoded2, err2 := base64.StdEncoding.DecodeString(encoded)
		if err2 != nil {
			// Not base64 at all, return original
			return encoded
		}
		return string(decoded2)
	}
	return string(decoded)
}

// encodeROT13Base64 encodes using ROT13+Base64 for compatibility with moustique_client
func encodeROT13Base64(plaintext string) string {
	if plaintext == "" {
		return ""
	}
	encoded := base64.StdEncoding.EncodeToString([]byte(plaintext))
	return rot13(encoded)
}

// rot13 applies ROT13 cipher to a string
func rot13(s string) string {
	result := make([]byte, len(s))
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c >= 'A' && c <= 'Z' {
			result[i] = 'A' + (c-'A'+13)%26
		} else if c >= 'a' && c <= 'z' {
			result[i] = 'a' + (c-'a'+13)%26
		} else {
			result[i] = c
		}
	}
	return string(result)
}

// decodeParams decodes all parameters in a url.Values map
func decodeParams(params map[string][]string) map[string]string {
	result := make(map[string]string)
	for key, values := range params {
		if len(values) > 0 {
			result[key] = decodeROT13Base64(values[0])
		}
	}
	return result
}

func GetFileVersion() (string, error) {
	// Hämta sökvägen till det körbara programmet (motsvarar $0 i Perl)
	exePath, err := os.Executable()
	if err != nil {
		return "", err
	}

	// Öppna filen (det körbara programmet)
	file, err := os.Open(exePath)
	if err != nil {
		return "", err
	}
	defer file.Close()

	// Beräkna MD5-hash
	hasher := md5.New()
	if _, err := io.Copy(hasher, file); err != nil {
		return "", err
	}

	// Returnera hex-sträng
	return hex.EncodeToString(hasher.Sum(nil)), nil
}
