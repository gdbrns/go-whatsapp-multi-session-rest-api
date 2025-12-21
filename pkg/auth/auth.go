package auth

import (
	"github.com/gdbrns/go-whatsapp-multi-session-rest-api/pkg/env"
)

// AdminSecretKey for admin API endpoints (/admin/*)
// REQUIRED: Application will panic if not set
var AdminSecretKey string

func init() {
	// ADMIN_SECRET_KEY is REQUIRED - app will panic if not configured
	AdminSecretKey = env.MustGetEnvString("ADMIN_SECRET_KEY")
}
