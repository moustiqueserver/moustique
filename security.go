package main

import (
	"net"
	"regexp"
	"strings"
)

// SecurityChecker handles IP-based access control
type SecurityChecker struct {
	allowedIPs    map[string]bool
	teliaRegex    *regexp.Regexp
	localNetRegex *regexp.Regexp
}

// NewSecurityChecker creates a new security checker
func NewSecurityChecker(allowedPeers []string) *SecurityChecker {
	// Convert slice to map for fast lookups
	allowedIPs := make(map[string]bool)
	for _, ip := range allowedPeers {
		allowedIPs[ip] = true
	}

	return &SecurityChecker{
		allowedIPs:    allowedIPs,
		teliaRegex:    regexp.MustCompile(`telia\.com$`),
		localNetRegex: regexp.MustCompile(`^192\.168\.`),
	}
}

// IsPeerAllowed checks if a peer IP is allowed
func (sc *SecurityChecker) IsPeerAllowed(peerHost string) bool {
	if peerHost == "" || peerHost == "UNKNOWN" {
		return false
	}

	// Localhost check
	if peerHost == "127.0.0.1" || peerHost == "::1" || peerHost == "localhost" {
		return true
	}

	// Local network check (fastest)
	if sc.localNetRegex.MatchString(peerHost) {
		return true
	}

	// Known allowed IPs
	if sc.allowedIPs[peerHost] {
		return true
	}

	// Tailscale IP check (100.x.x.x range)
	if strings.HasPrefix(peerHost, "100.") {
		return sc.isTailscaleIP(peerHost)
	}

	return false
}

func (sc *SecurityChecker) isTailscaleIP(ip string) bool {
	parsedIP := net.ParseIP(ip)
	if parsedIP == nil {
		return false
	}

	// Check if it's in 100.64.0.0/10 range (Tailscale CGNAT)
	_, tailscaleNet, _ := net.ParseCIDR("100.64.0.0/10")
	if tailscaleNet != nil && tailscaleNet.Contains(parsedIP) {
		return true
	}

	return false
}
