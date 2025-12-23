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
func NewSecurityChecker() *SecurityChecker {
	return &SecurityChecker{
		allowedIPs: map[string]bool{
			"195.67.22.146":  true,
			"178.174.128.24": true,
			"100.91.1.30":    true,
			"100.127.172.70": true,
			"100.106.122.48": true,
			"100.76.232.85":  true,
			"94.191.136.145": true,
			"94.191.137.51":  true,
			"94.191.137.181": true,
			"79.136.65.91":   true,
		},
		teliaRegex:    regexp.MustCompile(`telia\.com$`),
		localNetRegex: regexp.MustCompile(`^192\.168\.`),
	}
}

// IsPeerAllowed checks if a peer IP is allowed
func (sc *SecurityChecker) IsPeerAllowed(peerHost string) bool {
	if peerHost == "" || peerHost == "UNKNOWN" {
		return false
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
