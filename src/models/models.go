package models

import (
	"time"
)

type InputUrl struct {
	URL         string     `json:"url" binding:"required,url"`
	CustomAlias string     `json:"custom_alias" binding:"omitempty,alphanum,min=3,max=6"`
	ExpiresAt   *time.Time `json:"expires_at" binding:"omitempty,gt=now"`
}

type Link struct {
	ID          int        `json:"id"`
	URL         string     `json:"url"`
	Code        string     `json:"code"`
	ExpiresAt   *time.Time `json:"expires_at"`
	VisitsCount int        `json:"visits_count"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
}

type Visit struct {
	ID        int       `json:"id"`
	LinkID    int       `json:"link_id"`
	IPAddress string    `json:"ip_address"`
	UserAgent string    `json:"user_agent"`
	Referrer  string    `json:"referrer"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}
