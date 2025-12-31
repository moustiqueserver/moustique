package main

import (
	"sync"
	"time"
)

// IPStats tracks request statistics per IP address
type IPStats struct {
	mu         sync.RWMutex
	requests   map[string]*IPRequestInfo // IP -> request info
}

// IPRequestInfo holds request information for an IP address
type IPRequestInfo struct {
	IP             string
	RequestCount   int64
	FirstSeen      int64
	LastSeen       int64
	LastSeenString string
}

// NewIPStats creates a new IP statistics tracker
func NewIPStats() *IPStats {
	return &IPStats{
		requests: make(map[string]*IPRequestInfo),
	}
}

// RecordRequest records a request from an IP address
func (s *IPStats) RecordRequest(ip string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now().Unix()
	nowStr := formatNiceDateTime(now)

	if info, exists := s.requests[ip]; exists {
		info.RequestCount++
		info.LastSeen = now
		info.LastSeenString = nowStr
	} else {
		s.requests[ip] = &IPRequestInfo{
			IP:             ip,
			RequestCount:   1,
			FirstSeen:      now,
			LastSeen:       now,
			LastSeenString: nowStr,
		}
	}
}

// GetAllIPs returns all tracked IP addresses with their statistics
func (s *IPStats) GetAllIPs() []*IPRequestInfo {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := make([]*IPRequestInfo, 0, len(s.requests))
	for _, info := range s.requests {
		// Create a copy to avoid race conditions
		result = append(result, &IPRequestInfo{
			IP:             info.IP,
			RequestCount:   info.RequestCount,
			FirstSeen:      info.FirstSeen,
			LastSeen:       info.LastSeen,
			LastSeenString: info.LastSeenString,
		})
	}

	return result
}

// GetIPInfo returns request information for a specific IP
func (s *IPStats) GetIPInfo(ip string) *IPRequestInfo {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if info, exists := s.requests[ip]; exists {
		// Return a copy
		return &IPRequestInfo{
			IP:             info.IP,
			RequestCount:   info.RequestCount,
			FirstSeen:      info.FirstSeen,
			LastSeen:       info.LastSeen,
			LastSeenString: info.LastSeenString,
		}
	}

	return nil
}
