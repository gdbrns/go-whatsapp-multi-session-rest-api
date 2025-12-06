package validation

import (
	"errors"
	"net/url"
	"regexp"
	"strings"
)

var (
	phonePattern = regexp.MustCompile(`^[1-9][0-9]{5,15}$`)
)

// ValidatePhone ensures international format (no leading 0, digits only, length 6-16).
func ValidatePhone(phone string) error {
	trimmed := strings.TrimSpace(phone)
	if trimmed == "" {
		return errors.New("phone number cannot be empty")
	}
	if strings.HasPrefix(trimmed, "+") {
		trimmed = trimmed[1:]
	}
	if strings.HasPrefix(trimmed, "0") {
		return errors.New("phone number must be in international format without leading 0")
	}
	if !phonePattern.MatchString(trimmed) {
		return errors.New("phone number must be digits only and at least 6 characters")
	}
	return nil
}

// ValidateChatJID ensures chat identifier is provided.
func ValidateChatJID(chatJID string) error {
	if strings.TrimSpace(chatJID) == "" {
		return errors.New("chat_jid is required")
	}
	return nil
}

// ValidateURL ensures a non-empty valid URL when provided.
func ValidateURL(raw string) error {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return errors.New("url cannot be empty")
	}
	if _, err := url.ParseRequestURI(raw); err != nil {
		return errors.New("url must be valid")
	}
	return nil
}

