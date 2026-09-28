package provider

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"net"
	"net/url"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"
)

const (
	maxNameRunes = 80
	maxNameBytes = 256
	maxURLBytes  = 2048
	idPrefix     = "prov_"
	idBytes      = 16
)

var (
	// ErrInvalidInput reports invalid provider configuration data.
	ErrInvalidInput = errors.New("provider: invalid input")
	// ErrInvalidID reports a malformed provider identifier.
	ErrInvalidID = errors.New("provider: invalid ID")
	// ErrNotFound reports a provider configuration that does not exist.
	ErrNotFound = errors.New("provider: not found")
	// ErrHasAccounts prevents deletion while accounts still reference a provider.
	ErrHasAccounts = errors.New("provider: has accounts")
)

func validateKind(kind string) error {
	if !knownKind(kind) {
		return fmt.Errorf("%w: unknown kind", ErrInvalidInput)
	}
	return nil
}

func validateName(value string) (string, error) {
	name := strings.TrimSpace(value)
	if name == "" || !utf8.ValidString(name) || len(name) > maxNameBytes || utf8.RuneCountInString(name) > maxNameRunes {
		return "", fmt.Errorf("%w: name length", ErrInvalidInput)
	}
	for _, r := range name {
		if unicode.IsControl(r) {
			return "", fmt.Errorf("%w: name contains control characters", ErrInvalidInput)
		}
	}
	return name, nil
}

func validateBaseURL(value string) (string, error) {
	baseURL := strings.TrimSpace(value)
	if baseURL == "" {
		return "", nil // an empty value delegates to the future provider adapter default
	}
	if len(baseURL) > maxURLBytes || !utf8.ValidString(baseURL) {
		return "", fmt.Errorf("%w: base URL length", ErrInvalidInput)
	}
	u, err := url.Parse(baseURL)
	if err != nil || u.Hostname() == "" || u.Opaque != "" || u.User != nil || u.Fragment != "" || u.RawQuery != "" || u.ForceQuery {
		return "", fmt.Errorf("%w: base URL format", ErrInvalidInput)
	}
	if port := u.Port(); port != "" {
		value, err := strconv.Atoi(port)
		if err != nil || value < 1 || value > 65535 {
			return "", fmt.Errorf("%w: base URL port", ErrInvalidInput)
		}
	}
	if u.Scheme != "https" {
		// API Key 只能通过 HTTPS 发送；本机 HTTP 端点用于用户自建的代理。
		ip := net.ParseIP(u.Hostname())
		if u.Scheme != "http" || !(strings.EqualFold(u.Hostname(), "localhost") || ip != nil && ip.IsLoopback()) {
			return "", fmt.Errorf("%w: base URL must use HTTPS or loopback HTTP", ErrInvalidInput)
		}
	}
	return baseURL, nil
}

func validateID(id string) error {
	if len(id) != len(idPrefix)+idBytes*2 || !strings.HasPrefix(id, idPrefix) {
		return ErrInvalidID
	}
	for _, c := range id[len(idPrefix):] {
		if c < '0' || c > '9' {
			if c < 'a' || c > 'f' {
				return ErrInvalidID
			}
		}
	}
	return nil
}

func newID() (string, error) {
	var random [idBytes]byte
	if _, err := rand.Read(random[:]); err != nil {
		return "", err
	}
	return idPrefix + hex.EncodeToString(random[:]), nil
}
