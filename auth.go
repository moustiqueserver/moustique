package main

import (
	"encoding/json"
)

// Auth handles authentication
type Auth struct {
	db *Database
}

// NewAuth creates a new auth handler
func NewAuth(db *Database) *Auth {
	return &Auth{db: db}
}

// CheckPassword verifies a password
func (a *Auth) CheckPassword(provided string) bool {
	if provided == "" {
		return false
	}

	if !a.db.HasValue("moustique_pwd") {
		return false
	}

	storedJSON, err := a.db.GetValue("moustique_pwd")
	if err != nil {
		return false
	}

	var msg Message
	if err := json.Unmarshal([]byte(storedJSON), &msg); err != nil {
		return false
	}

	return provided == msg.Message
}
