package auth

import (
	"github.com/gdbrns/go-whatsapp-multi-session-rest-api/pkg/env"
)

// AdminSecretKey for admin API endpoints (/admin/*)
var AdminSecretKey string

func init() {
	AdminSecretKey, _ = env.GetEnvString("ADMIN_SECRET_KEY")
}
