package env

import (
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	_ "github.com/joho/godotenv/autoload"
)

// =============================================================================
// Required Environment Variables (will panic if not set)
// =============================================================================

// MustGetEnvString panics if the env var is not set - use for required secrets
func MustGetEnvString(envName string) string {
	v, err := GetEnvString(envName)
	if err != nil {
		panic(fmt.Sprintf("REQUIRED environment variable missing or empty: %s", envName))
	}
	return v
}

// =============================================================================
// Environment Variables with Defaults (safe for optional config)
// =============================================================================

// GetEnvStringOrDefault returns the env value or a default if not set
func GetEnvStringOrDefault(envName, defaultValue string) string {
	v, err := GetEnvString(envName)
	if err != nil {
		return defaultValue
	}
	return v
}

// GetEnvBoolOrDefault returns the env value or a default if not set
func GetEnvBoolOrDefault(envName string, defaultValue bool) bool {
	v, err := GetEnvBool(envName)
	if err != nil {
		return defaultValue
	}
	return v
}

// GetEnvIntOrDefault returns the env value or a default if not set
func GetEnvIntOrDefault(envName string, defaultValue int) int {
	v, err := GetEnvInt(envName)
	if err != nil {
		return defaultValue
	}
	return v
}

// GetEnvDurationOrDefault returns the env value as duration or a default if not set
func GetEnvDurationOrDefault(envName string, defaultValue time.Duration) time.Duration {
	v, err := GetEnvString(envName)
	if err != nil {
		return defaultValue
	}
	d, err := time.ParseDuration(v)
	if err != nil {
		return defaultValue
	}
	return d
}

// =============================================================================
// Core Environment Variable Getters
// =============================================================================

func SanitizeEnv(envName string) (string, error) {
	if len(envName) == 0 {
		return "", errors.New("Environment Variable Name Should Not Empty")
	}

	retValue := strings.TrimSpace(os.Getenv(envName))
	if len(retValue) == 0 {
		return "", errors.New("Environment Variable '" + envName + "' Has an Empty Value")
	}

	return retValue, nil
}

func GetEnvString(envName string) (string, error) {
	envValue, err := SanitizeEnv(envName)
	if err != nil {
		return "", err
	}

	return envValue, nil
}

func GetEnvBool(envName string) (bool, error) {
	envValue, err := SanitizeEnv(envName)
	if err != nil {
		return false, err
	}

	retValue, err := strconv.ParseBool(envValue)
	if err != nil {
		return false, err
	}

	return retValue, nil
}

func GetEnvInt(envName string) (int, error) {
	envValue, err := SanitizeEnv(envName)
	if err != nil {
		return 0, err
	}

	retValue, err := strconv.ParseInt(envValue, 0, 0)
	if err != nil {
		return 0, err
	}

	return int(retValue), nil
}

func GetEnvFloat32(envName string) (float32, error) {
	envValue, err := SanitizeEnv(envName)
	if err != nil {
		return 0, err
	}

	retValue, err := strconv.ParseFloat(envValue, 32)
	if err != nil {
		return 0, err
	}

	return float32(retValue), nil
}

func GetEnvFloat64(envName string) (float64, error) {
	envValue, err := SanitizeEnv(envName)
	if err != nil {
		return 0, err
	}

	retValue, err := strconv.ParseFloat(envValue, 64)
	if err != nil {
		return 0, err
	}

	return float64(retValue), nil
}
