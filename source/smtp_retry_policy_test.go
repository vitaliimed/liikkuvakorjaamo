package main

import (
	"errors"
	"fmt"
	"testing"
)

func TestBrevo525SMTPIPBlockingStopsAutomaticTaskRetry(t *testing.T) {
	tests := []struct {
		name   string
		err    error
		expect bool
	}{
		{"brevo smtp auth blocked", errors.New(`525 "5.7.1 Unauthorized IP address"`), true},
		{"wrapped permanent denial", fmt.Errorf("confirmation: %w", errors.New(`525 5.7.1 Unauthorized IP address`)), true},
		{"normal success", nil, false},
		{"authentication wrong password", errors.New("535 5.7.8 Authentication failed"), false},
		{"temporary service error", errors.New("451 SMTP temporarily unavailable"), false},
		{"network failure", errors.New("connection reset by peer"), false},
		{"other permission issue", errors.New("525 without policy message"), false},
		{"other message containing unauthorized", errors.New("403 unauthorized ip address"), false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := isSMTPIPAuthorizationDenied(tc.err); got != tc.expect {
				t.Fatalf("isSMTPIPAuthorizationDenied(%v) = %v; want %v", tc.err, got, tc.expect)
			}
		})
	}
}
