package main

import (
	"net/mail"
	"net/url"
	"regexp"
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

// mail.ParseAddress accepts some RFC-valid characters that have a special
// meaning in a mailto URL (such as '?' introducing a query). For a public
// contact button, accept only plain, common email mailbox syntax.
var thankYouPlainMailbox = regexp.MustCompile(`^[A-Za-z0-9._+-]+@[A-Za-z0-9](?:[A-Za-z0-9-]*[A-Za-z0-9])?(?:\.[A-Za-z0-9](?:[A-Za-z0-9-]*[A-Za-z0-9])?)+package main

import (
	"net/mail"
	"net/url"
	"regexp"
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

)

// The canonical service mailbox remains visible when the admin has not yet
// configured a contact address. Invalid values never enter a mailto link.
func thankYouContactEmail(raw string) string {
	const fallback = "service@liikkuvakorjaamo.fi"
	value := strings.TrimSpace(raw)
	if value == "" {
		return fallback
	}
	address, err := mail.ParseAddress(value)
	if err != nil || address.Address != value || !thankYouPlainMailbox.MatchString(value) {
		return fallback
	}
	return value
}
