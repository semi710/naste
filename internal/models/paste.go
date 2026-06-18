package models

import "time"

// Paste represents a stored paste with its metadata.
type Paste struct {
	Slug      string    `json:"slug"`
	Private   bool      `json:"private"`
	Lang      string    `json:"lang"`
	CreatedAt time.Time `json:"created_at"`
	Size      int64     `json:"size"`
	TTL       string    `json:"ttl"`
}
