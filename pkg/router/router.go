package router

import (
	"strconv"
	"strings"

	"github.com/gdbrns/go-whatsapp-multi-session-rest-api/pkg/env"
)

var BaseURL, CORSOrigin, BodyLimit string
var GZipLevel int
var CacheTTLSeconds int
var bodyLimitBytes int

func init() {
	// HTTP_BASE_URL: empty by default (no prefix)
	BaseURL = env.GetEnvStringOrDefault("HTTP_BASE_URL", "")
	BaseURL = strings.TrimSpace(BaseURL)
	BaseURL = strings.TrimRight(BaseURL, "/")
	if BaseURL != "" && BaseURL != "/" {
		BaseURL = "/" + strings.TrimLeft(BaseURL, "/")
	} else {
		BaseURL = ""
	}

	// HTTP_CORS_ORIGIN: default "*" (allow all)
	CORSOrigin = env.GetEnvStringOrDefault("HTTP_CORS_ORIGIN", "*")

	// HTTP_BODY_LIMIT_SIZE: default "8M"
	BodyLimit = env.GetEnvStringOrDefault("HTTP_BODY_LIMIT_SIZE", "8M")
	bodyLimitBytes = parseBodyLimit(BodyLimit)

	// HTTP_GZIP_LEVEL: default 1
	GZipLevel = env.GetEnvIntOrDefault("HTTP_GZIP_LEVEL", 1)

	// HTTP_CACHE_TTL_SECONDS: default 5
	CacheTTLSeconds = env.GetEnvIntOrDefault("HTTP_CACHE_TTL_SECONDS", 5)
}

func BodyLimitBytes() int {
	return bodyLimitBytes
}

func parseBodyLimit(limit string) int {
	const defaultLimit = 8 * 1024 * 1024
	limit = strings.TrimSpace(strings.ToUpper(limit))
	if limit == "" {
		return defaultLimit
	}
	multiplier := 1
	switch {
	case strings.HasSuffix(limit, "K"):
		multiplier = 1024
		limit = strings.TrimSuffix(limit, "K")
	case strings.HasSuffix(limit, "M"):
		multiplier = 1024 * 1024
		limit = strings.TrimSuffix(limit, "M")
	case strings.HasSuffix(limit, "G"):
		multiplier = 1024 * 1024 * 1024
		limit = strings.TrimSuffix(limit, "G")
	}
	value, err := strconv.Atoi(strings.TrimSpace(limit))
	if err != nil || value <= 0 {
		return defaultLimit
	}
	return value * multiplier
}
