package router

import (
	"strconv"
	"strings"

	"github.com/gdbrns/go-whatsapp-multi-session-rest-api/pkg/env"
)

var BaseURL, CORSOrigin, BodyLimit string
var GZipLevel int
var CacheCapacity, CacheTTLSeconds int
var bodyLimitBytes int

func init() {
	var err error

	BaseURL, err = env.GetEnvString("HTTP_BASE_URL")
	if err != nil {
		BaseURL = ""
	} else {
		BaseURL = strings.TrimSpace(BaseURL)
		BaseURL = strings.TrimRight(BaseURL, "/")
		if BaseURL == "" || BaseURL == "/" {
			BaseURL = ""
		} else {
			BaseURL = "/" + strings.TrimLeft(BaseURL, "/")
		}
	}

	CORSOrigin, err = env.GetEnvString("HTTP_CORS_ORIGIN")
	if err != nil {
		CORSOrigin = "*"
	}

	BodyLimit, err = env.GetEnvString("HTTP_BODY_LIMIT_SIZE")
	if err != nil {
		BodyLimit = "8M"
	}
	bodyLimitBytes = parseBodyLimit(BodyLimit)

	GZipLevel, err = env.GetEnvInt("HTTP_GZIP_LEVEL")
	if err != nil {
		GZipLevel = 1
	}

	CacheCapacity, err = env.GetEnvInt("HTTP_CACHE_CAPACITY")
	if err != nil {
		CacheCapacity = 100
	}

	CacheTTLSeconds, err = env.GetEnvInt("HTTP_CACHE_TTL_SECONDS")
	if err != nil {
		CacheTTLSeconds = 5
	}
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
