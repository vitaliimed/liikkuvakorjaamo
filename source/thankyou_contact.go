package main

import (
	"net/mail"
	"net/url"
	"strings"
)

// Only publish a real Instagram HTTPS profile link. Do not render invalid or
// unconfigured social links as clickable client-facing calls to action.
func thankYouInstagramURL(raw string) string {
	value := strings.TrimSpace(raw)
	u, err := url.Parse(value)
	if err != nil || u.Scheme != "https" || u.User != nil || u.Opaque != "" ||
		u.Port() != "" || (u.Hostname() != "instagram.com" && u.Hostname() != "www.instagram.com") ||
		strings.Trim(u.Path, "/") == "" {
		return ""
	}
	return u.String()
}

// The canonical service mailbox remains visible when the admin has not yet
// configured a contact address. Invalid values never enter a mailto link.
func thankYouContactEmail(raw string) string {
	const fallback = "service@liikkuvakorjaamo.fi"
	value := strings.TrimSpace(raw)
	if value == "" {
		return fallback
	}
	address, err := mail.ParseAddress(value)
	if err != nil || address.Address != value {
		return fallback
	}
	return value
}
