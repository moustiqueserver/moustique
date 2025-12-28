package main

import (
	"sync"
	"time"
)

// RateLimiter tracks request rates per user
type RateLimiter struct {
	mu            sync.RWMutex
	userLimits    map[string]int           // username -> requests per minute (0 = unlimited)
	userRequests  map[string][]int64       // username -> timestamps of requests in current minute
	defaultLimit  int                       // default requests per minute
}

// NewRateLimiter creates a new rate limiter
func NewRateLimiter(defaultLimit int) *RateLimiter {
	return &RateLimiter{
		userLimits:   make(map[string]int),
		userRequests: make(map[string][]int64),
		defaultLimit: defaultLimit,
	}
}

// SetUserLimit sets a custom rate limit for a user (0 = unlimited)
func (rl *RateLimiter) SetUserLimit(username string, limit int) {
	rl.mu.Lock()
	defer rl.mu.Unlock()
	rl.userLimits[username] = limit
}

// GetUserLimit gets the rate limit for a user
func (rl *RateLimiter) GetUserLimit(username string) int {
	rl.mu.RLock()
	defer rl.mu.RUnlock()
	if limit, ok := rl.userLimits[username]; ok {
		return limit
	}
	return rl.defaultLimit
}

// AllowRequest checks if a user is allowed to make a request
func (rl *RateLimiter) AllowRequest(username string) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	// Get limit for this user
	limit := rl.defaultLimit
	if userLimit, ok := rl.userLimits[username]; ok {
		limit = userLimit
	}

	// 0 = unlimited
	if limit == 0 {
		return true
	}

	now := time.Now().Unix()
	oneMinuteAgo := now - 60

	// Get or create request list for user
	requests, exists := rl.userRequests[username]
	if !exists {
		requests = []int64{}
	}

	// Remove requests older than 1 minute
	validRequests := []int64{}
	for _, ts := range requests {
		if ts > oneMinuteAgo {
			validRequests = append(validRequests, ts)
		}
	}

	// Check if under limit
	if len(validRequests) >= limit {
		rl.userRequests[username] = validRequests
		return false
	}

	// Add current request
	validRequests = append(validRequests, now)
	rl.userRequests[username] = validRequests

	return true
}

// GetUserRequestCount returns the number of requests in the last minute for a user
func (rl *RateLimiter) GetUserRequestCount(username string) int {
	rl.mu.RLock()
	defer rl.mu.RUnlock()

	requests, exists := rl.userRequests[username]
	if !exists {
		return 0
	}

	now := time.Now().Unix()
	oneMinuteAgo := now - 60

	count := 0
	for _, ts := range requests {
		if ts > oneMinuteAgo {
			count++
		}
	}

	return count
}

// Cleanup removes old request data (call periodically)
func (rl *RateLimiter) Cleanup() {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now().Unix()
	oneMinuteAgo := now - 60

	for username, requests := range rl.userRequests {
		validRequests := []int64{}
		for _, ts := range requests {
			if ts > oneMinuteAgo {
				validRequests = append(validRequests, ts)
			}
		}

		if len(validRequests) == 0 {
			delete(rl.userRequests, username)
		} else {
			rl.userRequests[username] = validRequests
		}
	}
}
